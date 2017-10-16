package connectsvc

import (
	. "protodef/pconnector"
	"github.com/go-redis/redis"
	"server/connector/internal/config"
	"time"
	"github.com/emirpasic/gods/trees/binaryheap"
	"container/list"
	"sync"
	"sync/atomic"
	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
)

type subscriber struct {
	redisSubClient *redis.Client
	timer *time.Timer
	roomQueues map[string]*redblacktree.Tree
	userQueues map[string]*list.List
	lock sync.RWMutex
	seq int64
}

type subscriberQueue struct {
	queue *list.List
	lock sync.RWMutex
}

var subscriberClient = &subscriber{
	timer : time.NewTimer(time.Second * 2),
	roomQueues: make(map[string]*redblacktree.Tree),
	userQueues: make(map[string]*list.List),
}

//订阅
func subscribe(s *service) {
	channelName1 := "router"
	//Deprecated
	channelName2 := "mychannel"

	if subscriberClient.redisSubClient == nil {
		subscriberClient.redisSubClient = redis.NewClient(&redis.Options {
			Addr:				config.Config.RedisPubSub.Address,
			Password:			config.Config.RedisPubSub.Password,
		})
	}

	pubsub := subscriberClient.redisSubClient.Subscribe(channelName1, channelName2)

	go func() {
		for {
			<- subscriberClient.timer.C
			subscriberClient.purge(s)
			subscriberClient.timer.Reset(time.Second * 2)
		}
	}()

	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			log.Error("Receive from channel, err=%s", err)
			continue
		}
		log.Info("Receive from channel, channel=%s, payload=%s", msg.Channel, msg.Payload)

		if msg.Channel == channelName1 || msg.Channel == channelName2 {
			subscriberClient.handleSubscription(msg.Payload, s)
		}
	}
}

func (subscriber *subscriber)purge(service *service)  {
	start := time.Now().UnixNano() / 1000000
	log.Error("100000 %d", time.Now().UnixNano() / 1000000 - start)

	func() {
		log.Error("150000 %d", time.Now().UnixNano() / 1000000 - start)
		subscriber.lock.Lock()
		log.Error("170000 %d", time.Now().UnixNano() / 1000000 - start)
		defer subscriber.lock.Unlock()

		log.Error("200000 %d", time.Now().UnixNano() / 1000000 - start)

		now := time.Now().UnixNano() / 1000000
		for _, queue := range subscriber.roomQueues {
			for quit := false; !queue.Empty() && !quit; {
				for it := queue.Iterator(); it.Next(); {
					m := it.Value().(*ToClientMessage)
					if now - m.Time > 2000 {
						queue.Remove(it.Key())
					} else {
						quit = true
					}
					break
				}
			}
		}
	}()

	log.Error("240000 %d", time.Now().UnixNano() / 1000000 - start)
	var outQueue int32 = subscriberClient.stat(service)

	log.Error("250000 %d, %d", time.Now().UnixNano() / 1000000 - start, outQueue)

	if outQueue > 10000 {
		subscriber.lock.Lock()
		defer subscriber.lock.Unlock()
		log.Error("270000 %d", time.Now().UnixNano() / 1000000 - start)

		log.Error("350000 %d", time.Now().UnixNano() / 1000000 - start)
		inverseIntComparator := func(a, b interface{}) int {
			q1 := a.(*redblacktree.Tree)
			q2 := b.(*redblacktree.Tree)
			return -(q1.Size() - q2.Size())
		}
		log.Error("360000 %d", time.Now().UnixNano() / 1000000 - start)
		var purgeHeap *binaryheap.Heap = binaryheap.NewWith(inverseIntComparator)

		log.Error("400000 %d", time.Now().UnixNano() / 1000000 - start)
		for _, queue := range subscriber.roomQueues {
			if queue.Size() > 0 {
				purgeHeap.Push(queue)
			}
		}

		first := true

		for ; outQueue > 10000; outQueue-- {
			c, ok := purgeHeap.Pop()
			if !ok {
				break
			}
			queue := c.(*redblacktree.Tree)
			node := queue.Left()
			if node != nil {
				m := node.Value.(*ToClientMessage)
				queue.Remove(node.Key)
				if first {
					log.Error("discard: %s, %s", m.MessageId, m.TimeText)
					first = false
				}
			}
			if queue.Size() > 0 {
				purgeHeap.Push(queue)
			}
		}
		log.Error("500000 %d", time.Now().UnixNano() / 1000000 - start)
	}
}

func (subscriber subscriber)handleSubscription(payload string, s *service)  {
	var fromRouterMessage FromRouterMessage
	fromRouterMessage.Unmarshal([]byte(payload))

	log.Trace("10000: %s", fromRouterMessage.TimeText)

	if fromRouterMessage.ToUserId != "" {
		subscriberClient.deliverToUser(s, fromRouterMessage)
	}
	if fromRouterMessage.ToRoomId != "" {
		subscriberClient.deliverToRoom(s, fromRouterMessage)
	}
}

func (subscriber *subscriber)deliverToUser(s *service, fromRouterMessage FromRouterMessage) {
	log.Info("deliver to user: %s", fromRouterMessage.ToUserId)

	subscriber.lock.Lock()
	defer subscriber.lock.Unlock()
	var queue, ok = subscriber.userQueues[fromRouterMessage.ToUserId]
	if !ok {
		queue = list.New()
		subscriber.userQueues[fromRouterMessage.ToUserId] = queue
	}
	toClientMessage := subscriber.convert(fromRouterMessage)
	queue.PushBack(&toClientMessage)
}

func (subscriber *subscriber)deliverToRoom(s *service, fromRouterMessage FromRouterMessage)  {
	log.Info("deliver to room: %s", fromRouterMessage.ToRoomId)

	subscriber.lock.Lock()
	defer subscriber.lock.Unlock()
	var queue, ok = subscriber.roomQueues[fromRouterMessage.ToRoomId]
	if !ok {
		queue = redblacktree.NewWith(utils.Int64Comparator)
		subscriber.roomQueues[fromRouterMessage.ToRoomId] = queue
	}
	toClientMessage := subscriber.convert(fromRouterMessage)
	queue.Put(atomic.AddInt64(&subscriber.seq, 1), &toClientMessage)
}

func (subscriber *subscriber) stat(service *service) int32 {
	subscriber.lock.RLock()
	defer subscriber.lock.RUnlock()

	var outQueue int32 = 0
	for _, queue := range subscriberClient.roomQueues {
		func() {
			outQueue += int32(queue.Size())
		}()
	}
	return outQueue
}

func (subscriber *subscriber)convert(fromRouterMessage FromRouterMessage) ToClientMessage {
	return ToClientMessage{
		Seq:		atomic.AddInt64(&subscriber.seq, 1),

		MessageId:	fromRouterMessage.MessageId,
		Time:		fromRouterMessage.Time,
		TimeText:	fromRouterMessage.TimeText,

		ToRoomId: 	fromRouterMessage.ToRoomId,
		ToUserId: 	fromRouterMessage.ToUserId,
		Params:   	fromRouterMessage.Params,

		RoomId:  	fromRouterMessage.ToRoomId,
		UserId:  	fromRouterMessage.ToUserId,
		Content: 	fromRouterMessage.Params["content"],
	}
}