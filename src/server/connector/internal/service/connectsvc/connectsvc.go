// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connectsvc

import (
	"github.com/someonegg/bdmsg"
	"golang.org/x/net/context"
	"net"
	"time"

	. "common/config"
	. "common/errdef"
	. "protodef/pconnector"
	"github.com/go-redis/redis"
	"sync/atomic"
	"github.com/Shopify/sarama"
	. "server/connector/internal/config"
	"sync"
	"fmt"
	"github.com/golang/protobuf/proto"
	"strconv"
)

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:9921",
	Password: "BrightHe0", // no password set
	DB:       0,  // use default DB
})

var producer sarama.SyncProducer
var producerInited = false

type service struct {
	*bdmsg.Server
	clientM *ClientManager
	roomM *RoomManager
}

var ops uint64 = 0
var ops2 uint64 = 0

func newService(l net.Listener, handshakeTO time.Duration, pumperInN, pumperOutN int, clientM *ClientManager, roomM *RoomManager) *service {

	s := &service{clientM: clientM, roomM: roomM}

	channelName1 := "router"
	//Deprecated
	channelName2 := "mychannel"
	pubsub := client.Subscribe(channelName1, channelName2)
	go func() {
		for {
			msg, err := pubsub.ReceiveMessage()
			if err != nil {
				log.Error("Receive from channel, err=%s", err)
				break
			}
			log.Info("Receive from channel, channel=%s, payload=%s", msg.Channel, msg.Payload)

			if msg.Channel == channelName1 || msg.Channel == channelName2 {
				var fromRouterMessage FromRouterMessage
				fromRouterMessage.Unmarshal([]byte(msg.Payload))
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
					treeSet, ok := s.roomM.clients[fromRouterMessage.ToRoomId]
					s.roomM.locker.Unlock()

					if ok {
						userIds := []string{}

						s.roomM.locker.Lock()
						treeSet.Each(func(index int, value interface{}) {
							userIds = append(userIds, value.(string))
						})
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
		}
	}()

	mux := bdmsg.NewPumpMux(nil)
	mux.HandleFunc(MsgTypeRegister, s.handleRegister)
	mux.HandleFunc(MsgTypeEnterRoom, s.handleEnterRoom)
	mux.HandleFunc(MsgTypeChat, s.handleMsg)

	s.Server = bdmsg.NewServerF(l, bdmsg.DefaultIOC, handshakeTO, mux, pumperInN, pumperOutN)

	ticker := time.NewTicker(time.Millisecond * 10)
	go func() {
		for range ticker.C {
			consume(0)
		}
	}()

	return s
}

/*
  客户端登记
 */
func (s *service) handleRegister(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	msc := p.UserData().(*bdmsg.SClient)
	if msc.Handshaked() {
		panic(ErrUnexpected)
	}

	var register Register
	err := register.Unmarshal(m) // unmarshal register
	if err != nil {
		panic(ErrParameter)
	}

	_, err = s.clientM.clientIn(register.UserId, register.Pass, msc, s.roomM)
	if err != nil {
		log.Error("handleRegister, err=%s", err)
		panic(ErrUnexpected)
	} else {
		log.Info("handleRegister, id=%s, remoteaddr=%s", register.UserId, msc.Conn().RemoteAddr())
	}

	// tell bdmsg that client is authorized
	msc.Handshake()
}


func (s *service) handleEnterRoom(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)

	var enterRoom EnterRoom
	err := enterRoom.Unmarshal(m) // unmarshal enterRoom
	if err != nil {
		panic(ErrParameter)
	}

	c.roomId = enterRoom.RoomId
	s.roomM.clientIn(c.ID, enterRoom.RoomId)
	log.Info("handleEnterRoom, id=%s, roomId=%s", c.ID, enterRoom.RoomId)
}

func (s *service) handleMsg(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)

	var roomId string = c.roomId
	var level int
	var nickname string
	var params map[string]string;


	log.Info("handleMsg, id=%s, t=%d, time=%d, m=%s", c.ID, t, time.Now().UnixNano() / 1000000, string(m[:]))

	switch t {
	case 1:
		var chat Chat
		err := chat.Unmarshal(m) // unmarshal chat
		if err != nil {
			panic(ErrParameter)
		}
		if chat.RoomId != "" {
			roomId = chat.RoomId
			s.roomM.clientIn(c.ID, roomId)
		}
		level = chat.Level
		nickname = chat.Nickname
		params = map[string]string{"content": chat.Content}
		break
	}

	var redisMsg = FromConnectorMessage{
		strconv.FormatUint(atomic.AddUint64(&ops, 1), 10),
		roomId,
		c.ID,
		nickname,
		time.Now().UnixNano() / 1000000.,
		int32(level),
		int32(t),
		params,
	}

	locker.Lock()
	if msgs[roomId] == nil {
		msgs[roomId] = NewRoomMsgToCompute()
	}
	msgs[roomId].Add(redisMsg)
	locker.Unlock()
}

var msgs = make(map[string]*RoomMsgToCompute)
var locker sync.RWMutex

func consume(id int) {
	i := atomic.AddUint64(&ops2, 1)
	consumeEvent(i, id)
}
func consumeEvent(i uint64, id int) {
	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			log.Error("err, %s", err) // 这里的err其实就是panic传入的内容，55
			print(err)
		}
	}()

	log.Debug("consume: <- produceEvt")
	readyToDeliver := []*FromConnectorMessage{}
	totalRestSize := 0
	start := time.Now().UnixNano() / 1000000
	log.Debug("consume, id=%d, event=%d, time=%d", id, i, start)
	func() {
		locker.RLock()
		defer locker.RUnlock()
		maxLength := 10000
		for k, v := range msgs {
			var  restSize int
			readyToDeliver, restSize = v.DrainTo(k, readyToDeliver, maxLength)
			totalRestSize += restSize
			log.Debug("size: %d", len(readyToDeliver))
		}
	}()
	deliver(readyToDeliver, totalRestSize, i, start)
}

func deliver(list []*FromConnectorMessage, totalRestSize int, i uint64, start int64) {
	if len(list) > 0 {
		msgs := FromConnectorMessages {
			Messages:list,
		}
		bytes, _ := proto.Marshal(&msgs)
		start = time.Now().UnixNano() / 1000000
		asyncProducer(bytes)
		end := time.Now().UnixNano() / 1000000
		log.Warn("finish consume, i=%d, time=%d, cost=%d, msgsLength=%d, textSize=%d, restSize=%d", i, end, end- start, len(list), len(bytes), totalRestSize)
	}
}

// asyncProducer 异步生产者
// 并发量大时，必须采用这种方式
func asyncProducer(bytes []byte) {
	config := sarama.NewConfig()

	config.Producer.MaxMessageBytes = 1024 * 1024 * 1024;
	config.Producer.RequiredAcks = sarama.NoResponse
	config.Producer.Partitioner = sarama.NewRandomPartitioner

	config.Producer.Timeout = 5 * time.Second
	//必须有这个选项
	config.Producer.Return.Successes = true
	config.Producer.Compression = sarama.CompressionGZIP
	producer, err := sarama.NewAsyncProducer(Config.Mq.KafkaBrokers, config)
	defer producer.Close()
	if err != nil {
		log.Error("%s", err)
		return
	}

	/*
	//必须有这个匿名函数内容
	go func(p sarama.AsyncProducer) {
		errors := p.Errors()
		success := p.Successes()
		for {
			select {
			case err := <-errors:
				if err != nil {
					log.Error("%s", err)
				}
			case <-success:
			}
		}
	}(producer)*/

	msg := &sarama.ProducerMessage{
		Topic: Config.Mq.Topic,
		Value: sarama.ByteEncoder(bytes),
	}
	producer.Input() <- msg
}

func Start(conf *BDMsgSvcConfT, clientM *ClientManager, roomM* RoomManager) (*bdmsg.Server, error) {
	l, err := net.ListenTCP("tcp", (*net.TCPAddr)(&conf.ListenAddr))
	if err != nil {
		log.Error("Start$net.ListenTCP, err=%s", err)
		return nil, ErrAddress
	}

	s := newService(l, time.Duration(conf.HandshakeTO), conf.InqueueN, conf.OutqueueN, clientM, roomM)
	s.Start()

	return s.Server, nil
}
