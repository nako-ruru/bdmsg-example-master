package connectsvc

import (
	"fmt"
	. "protodef/pconnector"
	"github.com/go-redis/redis"
	"server/connector/internal/config"
)

type subscriber struct {
	redisSubClient *redis.Client
}

var subscriberClient = subscriber{
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

	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			log.Error("Receive from channel, err=%s", err)
			pubsub.Close()
			go subscribe(s)
			break
		}
		log.Info("Receive from channel, channel=%s, payload=%s", msg.Channel, msg.Payload)

		if msg.Channel == channelName1 || msg.Channel == channelName2 {
			handleSubscription(msg.Payload, s)
		}
	}
}

func handleSubscription(payload string, s *service)  {
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

func (subscriber subscriber)deliverToUser(s *service, fromRouterMessage FromRouterMessage) {
	log.Info("deliver to user: %s", fromRouterMessage.ToUserId)
	toClientMessage := subscriberClient.convert(fromRouterMessage)
	subscriberClient.deliverToSingleClient(s, fromRouterMessage.ToUserId, &toClientMessage)
}

func (subscriber subscriber)deliverToRoom(s *service, fromRouterMessage FromRouterMessage)  {
	log.Info("deliver to room: %s", fromRouterMessage.ToRoomId)

	s.roomM.locker.Lock()
	userIdsWrapper, ok := s.roomM.clients[fromRouterMessage.ToRoomId]
	s.roomM.locker.Unlock()

	log.Trace("20000: %s", fromRouterMessage.TimeText)
	if ok {
		userIds := []string{}

		s.roomM.locker.Lock()
		for userId, _ := range userIdsWrapper {
			userIds = append(userIds, userId)
		}

		log.Trace("30000: %s", fromRouterMessage.TimeText)
		s.roomM.locker.Unlock()

		more := ""
		totalSize := len(userIds)
		if totalSize > 20 {
			more = "..."
		}
		log.Trace("40000: %s", fromRouterMessage.TimeText)
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

func (subscriber subscriber) deliverToSingleClient(service *service, userId string, m *ToClientMessage)  {
	var client *Client
	var ok bool
	func() {
		service.clientM.locker.Lock()
		defer service.clientM.locker.Unlock()
		client, ok = service.clientM.clients[userId]
	}()
	log.Trace("80000: %s", m.TimeText)

	if ok {
		client.ServerHello(m)
	} else {
		log.Warn("not found client: %s", userId)
	}
}

func (subscriber subscriber) stat(service *service) int32 {
	service.clientM.locker.RLock()
	defer service.clientM.locker.RUnlock()

	var outQueue int32 = 0
	for _, client := range service.clientM.clients {
		func() {
			client.queueLock.Lock()
			defer client.queueLock.Unlock()
			outQueue += int32(client.queue.Len())
		}()
	}
	return outQueue
}

func (subscriber subscriber)convert(fromRouterMessage FromRouterMessage) ToClientMessage {
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