package connectsvc

import (
	."protodef/pconnector"
	"server/connector/internal/config"
	"time"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/satori/go.uuid"
	"runtime/debug"
	"container/list"
	"sync"
)

//订阅
func subscribe(s *service) {
	groupID := uuid.NewV4().String()
	conf := cluster.NewConfig()
	conf.Consumer.Retry.Backoff = 200 * time.Millisecond
	conf.Consumer.Offsets.CommitInterval = 1 * time.Second
	conf.Consumer.Return.Errors = true
	conf.Consumer.Offsets.Initial = sarama.OffsetNewest
	conf.Group.Mode = cluster.ConsumerModePartitions
	conf.Group.Return.Notifications = true
	conf.Version = sarama.V0_10_1_0


	c, err := cluster.NewConsumer(config.Config.Mq.KafkaBrokers, groupID, []string{config.Config.Mq.Topic}, conf)
	if err != nil {
		log.Error("Failed open consumer: %v\r\n%s", err, debug.Stack())
		return
	}

	defer c.Close()

	go func() {
		for err := range c.Errors() {
			log.Error("Error: %s\r\n%s", err.Error(), debug.Stack())
		}
	}()

	//go func() {
	//	for note := range c.Notifications() {
	//		log.Info("Rebalanced: %+v", note)
	//	}
	//}()

	for {
		select {
		case part, ok := <-c.Partitions():
			if !ok {
				return
			}

			// start a separate goroutine to consume messages
			go func(pc cluster.PartitionConsumer) {
				defer pc.Close()

				for {
					select {
					case msg := <-pc.Messages():
						if msg != nil {
							log.Info("%s/%d/%d\t%s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
							c.MarkOffset(msg, "") //MarkOffset 并不是实时写入kafka，有可能在程序crash时丢掉未提交的offset
							handleSubscription(fmt.Sprintf("%s", msg.Value), s)
						}
					}
				}
			}(part)
		}
	}
}

func handleSubscription(payload string, s *service) {
	var fromRouterMessage FromRouterMessage
	fromRouterMessage.Unmarshal([]byte(payload))
	start := time.Now().UnixNano() / 1000000
	log.Debug("1000: %d", time.Now().UnixNano() / 1000000 - start)
	if fromRouterMessage.ToUserId != "" {
		log.Info("found client, userId=%s", fromRouterMessage.ToUserId)
		toClientMessage := ToClientMessage{
			ToRoomId: fromRouterMessage.ToRoomId,
			ToUserId: fromRouterMessage.ToUserId,
			Params:   fromRouterMessage.Params,

			RoomId:  fromRouterMessage.ToRoomId,
			UserId:  fromRouterMessage.ToUserId,
			Content: fromRouterMessage.Params["content"],
		}
		deliver2(fromRouterMessage.ToUserId, toClientMessage)
	}
	if fromRouterMessage.ToRoomId != "" {
		log.Debug("2000: %d", time.Now().UnixNano()/1000000-start)
		s.roomM.locker.Lock()
		userIdsWrapper, ok := s.roomM.clients[fromRouterMessage.ToRoomId]
		s.roomM.locker.Unlock()
		log.Debug("3000: %d", time.Now().UnixNano()/1000000-start)

		if ok {
			userIds := []string{}

			log.Debug("4000: %d", time.Now().UnixNano()/1000000-start)
			s.roomM.locker.Lock()
			for userId, _ := range userIdsWrapper {
				userIds = append(userIds, userId)
			}
			log.Debug("5000: %d", time.Now().UnixNano()/1000000-start)

			s.roomM.locker.Unlock()

			log.Debug("6000: %d", time.Now().UnixNano()/1000000-start)
			more := ""
			totalSize := len(userIds)
			if totalSize > 20 {
				more = "..."
			}
			log.Debug("7000: %d", time.Now().UnixNano()/1000000-start)
			var userIdsText string
			if totalSize == 0 {
				userIdsText = "[]"
			} else if totalSize <= 20 {
				userIdsText = fmt.Sprintf("%s", userIds)
			} else {
				userIdsText = fmt.Sprintf("%s", userIds[:20])
			}
			log.Info("found client, roomId=%s, totalSize=%d, userIds=%s%s", fromRouterMessage.ToRoomId, totalSize, userIdsText, more)

			log.Debug("8000: %d", time.Now().UnixNano()/1000000-start)
			toClientMessage := ToClientMessage{
				ToRoomId: fromRouterMessage.ToRoomId,
				ToUserId: fromRouterMessage.ToUserId,
				Params:   fromRouterMessage.Params,

				RoomId:  fromRouterMessage.ToRoomId,
				UserId:  fromRouterMessage.ToUserId,
				Content: fromRouterMessage.Params["content"],
			}
			log.Debug("9000: %d", time.Now().UnixNano()/1000000-start)
			for _, value := range userIds {
				log.Debug("9100: %d", time.Now().UnixNano()/1000000-start)
				deliver2(value, toClientMessage)
				log.Error("9700: %d", time.Now().UnixNano()/1000000-start)
			}
			log.Debug("10000: %d", time.Now().UnixNano()/1000000-start)
		}
	}
}

func deliver2(userId string, toClientMessage ToClientMessage) {
	pollLock.Lock()
	defer pollLock.Unlock()
	elements, ok := readyToPush[userId]
	if !ok {
		elements = list.New()
		readyToPush[userId] = elements
		elements.PushBack(toClientMessage)
	}
}

var readyToPush = make(map[string]*list.List)
var pollLock sync.RWMutex


func initSubscribeConsumer(s *service) {
	ticker := time.NewTicker(time.Millisecond * 50)
	go func() {
		for range ticker.C {
			consume2(s)
		}
	}()
}

func consume2(s *service) {
	var temp = make(map[string]*list.List)
	totalCount := 0
	func() {
		pollLock.Lock()
		defer pollLock.Unlock()
		for userId, messages := range readyToPush {
			temp[userId] = messages
			delete(readyToPush, userId)
			if totalCount >= 10000 {
				break
			}
		}
	}()

	for userId, messages := range temp {
		s.clientM.locker.Lock()
		client, ok := s.clientM.clients[userId]
		s.clientM.locker.Unlock()
		if ok {
			var array = []ToClientMessage{}
			for e := messages.Front(); e != nil; e = e.Next() {
				array = append(array, e.Value.(ToClientMessage))
			}
			client.ServerHello(array)
		}
	}
}