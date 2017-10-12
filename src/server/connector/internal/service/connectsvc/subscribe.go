package connectsvc

import (
	."protodef/pconnector"
	"github.com/go-redis/redis"
	"server/connector/internal/config"
	"time"
	"fmt"
	"container/list"
	"sync"
	"encoding/json"
)

var redisSubClient *redis.Client
//订阅
func subscribe(s *service) {
	channelName1 := "router"
	//Deprecated
	channelName2 := "mychannel"

	if redisSubClient == nil {
		redisSubClient = redis.NewClient(&redis.Options {
			Addr:				config.Config.RedisPubSub.Address,
			Password:			config.Config.RedisPubSub.Password,
		})
	}

	pubsub := redisSubClient.Subscribe(channelName1, channelName2)

	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			log.Error("Receive from channel, err=%s", err)
			break
		}
		log.Info("Receive from channel, channel=%s, payload=%s", msg.Channel, msg.Payload)

		if msg.Channel == channelName1 || msg.Channel == channelName2 {
			handleSubscription(msg.Payload, s)
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
			MessageId:fromRouterMessage.MessageId,
			Time:fromRouterMessage.Time,
			TimeText:fromRouterMessage.TimeText,

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
		toClientMessage := ToClientMessage{
			MessageId:fromRouterMessage.MessageId,
			Time:fromRouterMessage.Time,
			TimeText:fromRouterMessage.TimeText,

			ToRoomId: fromRouterMessage.ToRoomId,
			ToUserId: fromRouterMessage.ToUserId,
			Params:   fromRouterMessage.Params,

			RoomId:  fromRouterMessage.ToRoomId,
			UserId:  fromRouterMessage.ToUserId,
			Content: fromRouterMessage.Params["content"],
		}
		deliver2(fromRouterMessage.ToRoomId, toClientMessage)
	}
}

func deliver2(roomId string, toClientMessage ToClientMessage) {
	pollLock.Lock()
	defer pollLock.Unlock()
	elements, ok := roomMessages[roomId]
	if !ok {
		elements = list.New()
		roomMessages[roomId] = elements
	}
	elements.PushBack(toClientMessage)
	for ; elements.Len() > 100; {
		for e := elements.Front(); e != nil; e = e.Next() {
			elements.Remove(e)
			var jsonText, _ = json.Marshal(e.Value.(ToClientMessage))
			log.Warn("discard: %s", jsonText)
			break
		}
	}
}

var roomMessages = make(map[string]*list.List)
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
	pollLock.Lock()
	var temp = roomMessages
	roomMessages = make(map[string]*list.List)
	pollLock.Unlock()

	beforeQueueSize := 0
	start := time.Now().UnixNano() / 1000000

	for _, messages := range temp {
		beforeQueueSize += messages.Len()
	}
	log.Error("100000: %d", time.Now().UnixNano() / 1000000 - start)
	log.Error("beforeQueueSize: %d", beforeQueueSize,)

	for roomId, messages := range temp {
		var array = []ToClientMessage{}
		for e := messages.Front(); e != nil; e = e.Next() {
			array = append(array, e.Value.(ToClientMessage))
		}

		log.Debug("2000: %d", time.Now().UnixNano()/1000000-start)
		s.roomM.locker.Lock()
		userIdsWrapper, ok := s.roomM.clients[roomId]
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
			log.Info("found client, roomId=%s, totalSize=%d, userIds=%s%s", roomId, totalSize, userIdsText, more)

			log.Debug("8000: %d", time.Now().UnixNano()/1000000-start)

			mr, _ := json.Marshal(array)
			compressed := DoZlibCompress(mr)
			log.Error("uncompressed: %d, compressed: %d", len(mr), len(compressed))
			log.Debug("9000: %d", time.Now().UnixNano()/1000000-start)

			for _, value := range userIds {
				log.Debug("9100: %d", time.Now().UnixNano()/1000000-start)
				s.clientM.clients[value].ServerHello(compressed)
				log.Debug("9700: %d", time.Now().UnixNano()/1000000-start)
			}
			log.Debug("10000: %d", time.Now().UnixNano()/1000000-start)
		}
	}
	log.Error("200000: %d", time.Now().UnixNano() / 1000000 - start)
}