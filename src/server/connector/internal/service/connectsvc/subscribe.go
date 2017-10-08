package connectsvc

import (
	"github.com/satori/go.uuid"
	"github.com/bsm/sarama-cluster"
	"github.com/Shopify/sarama"
	"time"
	"runtime/debug"
	"fmt"
	"server/connector/internal/config"
	. "protodef/pconnector"
)

//订阅
func subscribe(s *service) {
	groupID := uuid.NewV4().String()
	conf := cluster.NewConfig()

	conf.Consumer.Return.Errors = true
	conf.Consumer.Offsets.Initial = sarama.OffsetNewest //初始从最新的offset开始
	conf.Group.Return.Notifications = true
	conf.Consumer.Offsets.CommitInterval = 1 * time.Second
	conf.Version = sarama.V0_10_2_0

	c, err := cluster.NewConsumer(config.Config.Mq.KafkaBrokers, groupID, []string{config.Config.Mq.Topic}, conf)
	if err != nil {
		log.Error("Failed open consumer: %v\r\n%s", err, debug.Stack())
		return
	}

	//defer c.Close()

	go func() {
		for err := range c.Errors() {
			log.Error("Error: %s\r\n%s", err.Error(), debug.Stack())
		}
	}()

	go func() {
		for note := range c.Notifications() {
			log.Info("Rebalanced: %+v", note)
		}
	}()

	for msg := range c.Messages() {
		log.Info("%s/%d/%d\t%s", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		c.MarkOffset(msg, "") //MarkOffset 并不是实时写入kafka，有可能在程序crash时丢掉未提交的offset
		handleSubscription(fmt.Sprintf("%s", msg.Value), s)
	}

	log.Error("heheheheheheheheheheh, %s", debug.Stack())
}

func handleSubscription(payload string, s *service)  {
	var fromRouterMessage FromRouterMessage
	fromRouterMessage.Unmarshal([]byte(payload))
	if fromRouterMessage.ToUserId != "" {
		s.clientM.locker.Lock()
		client, ok := s.clientM.clients[fromRouterMessage.ToUserId]
		s.clientM.locker.Unlock()
		if ok {
			log.Info("found client, userId=%s", fromRouterMessage.ToUserId)
			toClientMessage := ToClientMessage {
				ToRoomId: fromRouterMessage.ToRoomId,
				ToUserId: fromRouterMessage.ToUserId,
				Params:   fromRouterMessage.Params,

				RoomId:  fromRouterMessage.ToRoomId,
				UserId:  fromRouterMessage.ToUserId,
				Content: fromRouterMessage.Params["content"],
			}
			client.ServerHello(toClientMessage)
		}
	}
	if fromRouterMessage.ToRoomId != "" {
		s.roomM.locker.Lock()
		userIdsWrapper, ok := s.roomM.clients[fromRouterMessage.ToRoomId]
		s.roomM.locker.Unlock()

		if ok {
			userIds := []string{}

			s.roomM.locker.Lock()
			for userId, _ := range userIdsWrapper {
				userIds = append(userIds, userId)
			}

			s.roomM.locker.Unlock()

			more := ""
			totalSize := len(userIds)
			if totalSize > 20 {
				more = "..."
			}
			var userIdsText string
			if totalSize == 0 {
				userIdsText = "[]"
			} else if totalSize <= 20 {
				userIdsText = fmt.Sprintf("%s", userIds)
			} else {
				userIdsText = fmt.Sprintf("%s", userIds[:20])
			}
			log.Info("found client, roomId=%s, totalSize=%d, userIds=%s%s", fromRouterMessage.ToRoomId, totalSize, userIdsText, more)

			toClientMessage := ToClientMessage{
				ToRoomId: fromRouterMessage.ToRoomId,
				ToUserId: fromRouterMessage.ToUserId,
				Params:   fromRouterMessage.Params,

				RoomId:  fromRouterMessage.ToRoomId,
				UserId:  fromRouterMessage.ToUserId,
				Content: fromRouterMessage.Params["content"],
			}
			for _, value := range userIds {
				s.clientM.locker.Lock()
				client := s.clientM.clients[value]
				s.clientM.locker.Unlock()
				client.ServerHello(toClientMessage)
			}
		}
	}
}

