package connectsvc

import (
	"fmt"
	. "protodef/pconnector"
	"github.com/go-redis/redis"
	"server/connector/internal/config"
	"time"
	"github.com/emirpasic/gods/trees/binaryheap"
	"runtime/debug"
)

type subscriber struct {
	redisSubClient *redis.Client
	timer *time.Timer
}

var subscriberClient = subscriber{
	timer : time.NewTimer(time.Second * 2),
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

	pubSub := subscriberClient.redisSubClient.Subscribe(channelName1, channelName2)

	go func() {
		for {
			<- subscriberClient.timer.C
			subscriberClient.purge(s)
			subscriberClient.timer.Reset(time.Second * 2)
		}
	}()

	for {
		msg, err := pubSub.ReceiveMessage()
		if err != nil {
			log.Error("Receive from channel, err=%s\r\n%s", err, debug.Stack())
			continue
		}
		log.Info("Receive from channel, channel=%s, payload=%s", msg.Channel, msg.Payload)

		if msg.Channel == channelName1 || msg.Channel == channelName2 {
			subscriberClient.handleSubscription(msg.Payload, s)
		}
	}
}


func (subscriber *subscriber)purge(service *service)  {
	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			log.Error("err, %s\r\n%s", err, debug.Stack()) // 这里的err其实就是panic传入的内容，55
			println(err)
		}
	}()

	start := time.Now().UnixNano() / 1000000
	log.Trace("100000 %d", time.Now().UnixNano() / 1000000 - start)

	var outQueue int32 = subscriber.stat(service)

	log.Trace("200000 %d", time.Now().UnixNano() / 1000000 - start)

	if outQueue > 10000 {
		topComparator := func(a, b interface{}) int {
			c1 := a.(*Client)
			c2 := b.(*Client)
			var m1, m2 *ToClientMessage
			if e := c1.queue.Front(); e != nil {
				m1 = e.Value.(*ToClientMessage)
			}
			if e := c2.queue.Front(); e != nil {
				m2 = e.Value.(*ToClientMessage)
			}
			importanceDiff := -(m1.Importance - m2.Importance)
			if importanceDiff != 0 {
				return importanceDiff
			}
			return c1.level - c2.level
		}
		var purgeHeap *binaryheap.Heap = binaryheap.NewWith(topComparator)

		service.clientM.locker.RLock()
		defer service.clientM.locker.RUnlock()

		for _, client := range service.clientM.clients {
			client.lock.Lock()
		}
		log.Trace("300000 %d", time.Now().UnixNano() / 1000000 - start)

		for _, client := range service.clientM.clients {
			if client.queue.Len() > 0 {
				purgeHeap.Push(client)
			}
		}
		log.Trace("400000 %d", time.Now().UnixNano() / 1000000 - start)

		first := true

		for ; outQueue > 10000; outQueue-- {
			c, ok := purgeHeap.Pop()
			if !ok {
				break
			}
			client := c.(*Client)
			if e := client.queue.Front(); e != nil {
				m := e.Value.(*ToClientMessage)
				client.queue.Remove(e)
				if first {
					log.Error("discard: %s, %s, %s", client.ID, m.MessageId, m.TimeText)
					first = false
				}
			}
			if client.queue.Len() > 0 {
				purgeHeap.Push(client)
			}
		}
		log.Trace("500000 %d", time.Now().UnixNano() / 1000000 - start)

		for _, client := range service.clientM.clients {
			client.lock.Unlock()
		}
		log.Trace("600000 %d", time.Now().UnixNano() / 1000000 - start)
	}
}

func (subscriber subscriber)handleSubscription(payload string, s *service)  {
	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			log.Error("err, %s\r\n%s", err, debug.Stack()) // 这里的err其实就是panic传入的内容，55
			println(err)
		}
	}()

	var fromRouterMessage FromRouterMessage
	fromRouterMessage.Unmarshal([]byte(payload))

	log.Trace("10000: %s", fromRouterMessage.TimeText)

	if fromRouterMessage.ToUserId != "" {
		subscriberClient.deliverToUser(s, fromRouterMessage)
	}
	if fromRouterMessage.ToRoomId == "world" {
		subscriberClient.deliverToWorld(s, fromRouterMessage)
	} else if fromRouterMessage.ToRoomId != "" {
		subscriberClient.deliverToRoom(s, fromRouterMessage)
	}
}

func (subscriber *subscriber)deliverToUser(s *service, fromRouterMessage FromRouterMessage) {
	log.Info("deliver to user: %s", fromRouterMessage.ToUserId)
	toClientMessage := subscriberClient.convert(fromRouterMessage)
	subscriberClient.deliverToSingleClient(s, fromRouterMessage.ToUserId, &toClientMessage)
}

func (subscriber *subscriber)deliverToRoom(s *service, fromRouterMessage FromRouterMessage)  {
	log.Info("deliver to room: %s", fromRouterMessage.ToRoomId)

	s.roomM.locker.Lock()
	userIdToClientMap, ok := s.roomM.clients[fromRouterMessage.ToRoomId]
	s.roomM.locker.Unlock()

	log.Trace("20000: %s", fromRouterMessage.TimeText)
	if ok {
		userIds := []string{}

		func() {
			s.roomM.locker.Lock()
			defer s.roomM.locker.Unlock()

			roomOwnerId := subscriber.roomOwnerIdResolver(fromRouterMessage.ToRoomId)
			if _, ok := userIdToClientMap[roomOwnerId]; ok {
				userIds = append(userIds, roomOwnerId)
			}
			for userId, _ := range userIdToClientMap {
				if userId != roomOwnerId {
					userIds = append(userIds, userId)
				}
			}
			log.Trace("30000: %s", fromRouterMessage.TimeText)
		}()

		totalSize := len(userIds)
		more := subscriber.ternaryIf(totalSize <= 20, "", "...")
		var userIdsText string
		if totalSize == 0 {
			userIdsText = "[]"
		} else if totalSize <= 20 {
			userIdsText = fmt.Sprintf("%s", userIds)
		} else {
			userIdsText = fmt.Sprintf("%s", userIds[:20])
		}
		log.Trace("50000: %s", fromRouterMessage.TimeText)

		log.Info("found following users in room(%s), totalSize=%d, userIds=%s%s", fromRouterMessage.ToRoomId, totalSize, userIdsText, more)

		toClientMessage := subscriber.convert(fromRouterMessage)
		log.Trace("60000: %s", fromRouterMessage.TimeText)
		for _, value := range userIds {
			log.Trace("70000: %s", fromRouterMessage.TimeText)
			subscriber.deliverToSingleClient(s, value, &toClientMessage)
		}
	}
}

func (subscriber *subscriber)deliverToWorld(s *service, fromRouterMessage FromRouterMessage) {
	log.Info("deliver to world")
	toClientMessage := subscriber.convert(fromRouterMessage)
	s.clientM.locker.RLock()
	defer s.clientM.locker.RUnlock()

	for _, client := range s.clientM.clients {
		client.ServerHello(&toClientMessage)
	}
}

func (subscriber *subscriber) deliverToSingleClient(service *service, userId string, m *ToClientMessage)  {
	var client *Client = service.clientM.Client(userId)
	log.Trace("80000: %s", m.TimeText)

	if client != nil {
		client.ServerHello(m)
	} else {
		log.Warn("client not found: %s", userId)
	}
}

func (subscriber *subscriber)roomOwnerIdResolver(roomId string) string {
	return roomId
}

func (subscriber *subscriber) ternaryIf(flag bool, v1 string, v2 string) string {
	if flag {
		return v1
	}
	return v2
}

func (subscriber *subscriber) stat(service *service) int32 {
	service.clientM.locker.RLock()
	defer service.clientM.locker.RUnlock()

	var outQueue int32 = 0
	for _, client := range service.clientM.clients {
		func() {
			client.lock.Lock()
			defer client.lock.Unlock()
			outQueue += int32(client.queue.Len())
		}()
	}
	return outQueue
}

func (subscriber *subscriber)convert(fromRouterMessage FromRouterMessage) ToClientMessage {
	return ToClientMessage{
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