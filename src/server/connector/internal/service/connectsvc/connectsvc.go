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
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/streadway/amqp"
	"github.com/satori/go.uuid"
	"github.com/Shopify/sarama"
	"fmt"
	. "server/connector/internal/config"
)

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:9921",
	Password: "BrightHe0", // no password set
	DB:       0,  // use default DB
})

func failOnError(err error, msg string) {
	if err != nil {
		ch = nil
		log.Error("failOnError, error=%s", err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

var q amqp.Queue
var ch *amqp.Channel

var producer sarama.SyncProducer
var producerInited = false

type service struct {
	*bdmsg.Server
	clientM *ClientManager
	roomM *RoomManager
}

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
					client, ok := s.clientM.clients[fromRouterMessage.ToUserId]
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
					treeSet, ok := s.roomM.clients[fromRouterMessage.ToRoomId]
					if ok {
						userIds := []string{}
						count := 0;
						treeSet.Select(func(index int, value interface{}) bool {
							count++
							return count <= 5
						}).Each(func(index int, value interface{}) {
							userIds = append(userIds, value.(string))
						})
						more := ""
						totalSize := treeSet.Size()
						if(totalSize > len(userIds)) {
							more = "..."
						}
						log.Info("found client, roomId=%s, totalSize=%d, userIds=%s%s", fromRouterMessage.ToRoomId, totalSize, userIds, more)
						treeSet.Each(func(index int, value interface{}) {
							toClientMessage := ToClientMessage{
								ToRoomId: fromRouterMessage.ToRoomId,
								ToUserId: fromRouterMessage.ToUserId,
								Params:   fromRouterMessage.Params,

								RoomId:  fromRouterMessage.ToRoomId,
								UserId:  fromRouterMessage.ToUserId,
								Content: fromRouterMessage.Params["content"],
							}
							client := s.clientM.clients[value.(string)]
							client.ServerHello(toClientMessage)
						})
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

	var redisMsg = ToComputeMessage{
		uuid.NewV4().String(),
		roomId,
		c.ID,
		nickname,
		level,
		int(t),
		params,
		time.Now().UnixNano() / 1000000,
	}
	var bytes,_ = json.Marshal(redisMsg)


	var sendDirectly = false
	var mqType = "kafka"
	if sendDirectly {
		pong, err := client.Ping().Result()
		if err != nil {
			log.Error("handleMsg, pong=%s, err=%s", pong, err)
		}

		client.RPush(roomId, bytes)
	} else if mqType == "radismq" {
		client.RPush("connector", bytes)
	} else if mqType == "kafka" {
		if !producerInited {
			config := sarama.NewConfig()
			config.Producer.RequiredAcks = sarama.WaitForAll
			config.Producer.Partitioner = sarama.NewRandomPartitioner
			config.Producer.Return.Successes = true
			config.Producer.Compression = sarama.CompressionGZIP

			var err error
			producer, err = sarama.NewSyncProducer(Config.KafkaBrokers, config)
			if err != nil {
				log.Error("%s", err)
				panic(err)
			}
			producerInited = true
		}

		msg := &sarama.ProducerMessage {
			Topic: 			"connector",
			Partition: 		int32(-1),
			Key:       		sarama.StringEncoder("key"),
			Value:			sarama.ByteEncoder(bytes),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Error("Send message Fail, %s, host=%s", err, Config.KafkaBrokers)
		}
		log.Trace("Partition = %d, offset=%d\n", partition, offset)
	} else {
		var err error
		if ch == nil {
			var conn *amqp.Connection
			conn, err = amqp.Dial(Config.RabbitMqUrl)
			failOnError(err, "Failed to connect to RabbitMQ")
			ch, err = conn.Channel()
			failOnError(err, "Failed to open a channel")

			q, err = ch.QueueDeclare(
				"connector", // name
				false,   // durable
				false,   // delete when unused
				false,   // exclusive
				false,   // no-wait
				nil,     // arguments
			)
			failOnError(err, "Failed to declare a queue")
		}

		body := string(bytes)
		err = ch.Publish(
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing {
				ContentType: "text/plain",
				Body:        []byte(body),
			})
		failOnError(err, "Failed to publish a message")
	}
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
