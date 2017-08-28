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
}

func newService(l net.Listener, handshakeTO time.Duration, pumperInN, pumperOutN int,
	clientM *ClientManager) *service {

	s := &service{clientM: clientM}

	pubsub := client.Subscribe("mychannel")
	go func() {
		for {
			msg, err := pubsub.ReceiveMessage()
			if err != nil {
				log.Error("Receive from channel, err=%s", err)
				break
			}
			log.Info("Receive from channel, channel=%s, payload=%s", msg.Channel, msg.Payload)
			var hello PushMsg
			hello.Unmarshal([]byte(msg.Payload))
			i, ok := s.clientM.clients[hello.UserId]
			if ok {
				log.Info("found client, userId=%s", hello.UserId)
				i.ServerHello(hello)
			}
		}
	}()

	mux := bdmsg.NewPumpMux(nil)
	mux.HandleFunc(MsgTypeRegister, s.handleRegister)
	mux.HandleFunc(MsgTypeEnterRoom, s.handleMsg)
	mux.HandleFunc(MsgTypeChat, s.handleMsg)
	mux.HandleFunc(MsgTypeSupport, s.handleMsg)
	mux.HandleFunc(MsgTypeSendGift, s.handleMsg)
	mux.HandleFunc(MsgTypeShare, s.handleMsg)
	mux.HandleFunc(MsgTypeLevelUp, s.handleMsg)

	s.Server = bdmsg.NewServerF(l, bdmsg.DefaultIOC, handshakeTO,
		mux, pumperInN, pumperOutN)
	return s
}

func (s *service) handleRegister(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	msc := p.UserData().(*bdmsg.SClient)
	if msc.Handshaked() {
		panic(ErrUnexpected)
	}

	var request Register
	err := request.Unmarshal(m) // unmarshal request
	if err != nil {
		panic(ErrParameter)
	}

	_, err = s.clientM.clientIn(request.UserId, request.Pass, msc)
	if err != nil {
		log.Error("handleRegister, err=%s", err)
		panic(ErrUnexpected)
	}

	// tell bdmsg that client is authorized
	msc.Handshake()
}

func (s *service) handleMsg(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)

	var roomId string
	var level int
	var nickname string
	var params map[string]string;


	log.Info("handleMsg, id=%s, t=%d, m=%s", c.ID, t, string(m[:]))

	switch t {
	case 1:
		var hello Chat
		err := hello.Unmarshal(m) // unmarshal hello
		if err != nil {
			panic(ErrParameter)
		}
		roomId = hello.RoomId
		level = hello.Level
		nickname = hello.Nickname
		params = map[string]string{"content": hello.Content}
		break
	case 2:
		var hello Support
		err := hello.Unmarshal(m)
		if err != nil {
			panic(ErrParameter)
		}
		roomId = hello.RoomId
		level = hello.Level
		nickname = hello.Nickname
		params = map[string]string{}
		break
	case 3:
		var hello SendGift
		err := hello.Unmarshal(m)
		if err != nil {
			panic(ErrParameter)
		}
		roomId = hello.RoomId
		level = hello.Level
		nickname = hello.Nickname
		params = map[string]string{"giftId": hello.GiftId}
		break
	case 4:
		var hello EnterRoom
		err := hello.Unmarshal(m)
		if err != nil {
			panic(ErrParameter)
		}
		roomId = hello.RoomId
		level = hello.Level
		nickname = hello.Nickname
		params = map[string]string{}
		break
	case 5:
		var hello Share
		err := hello.Unmarshal(m)
		if err != nil {
			panic(ErrParameter)
		}
		roomId = hello.RoomId
		level = hello.Level
		nickname = hello.Nickname
		params = map[string]string{}
		break
	case 6:
		var hello LevelUp
		err := hello.Unmarshal(m) // unmarshal hello
		if err != nil {
			panic(ErrParameter)
		}
		roomId = hello.RoomId
		level = hello.Level
		nickname = hello.Nickname
		params = map[string]string{"level": fmt.Sprint(hello.Level)}
		break
	}

	var redisMsg = RedisMsg{uuid.NewV4().String(),roomId, c.ID, time.Now().UnixNano() / 1000000, int(t), nickname, level, params}
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
			log.Error("Send message Fail, %s", err)
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

func Start(conf *BDMsgSvcConfT, clientM *ClientManager) (*bdmsg.Server, error) {
	l, err := net.ListenTCP("tcp", (*net.TCPAddr)(&conf.ListenAddr))
	if err != nil {
		log.Error("Start$net.ListenTCP, err=%s", err)
		return nil, ErrAddress
	}

	s := newService(l, time.Duration(conf.HandshakeTO), conf.InqueueN, conf.OutqueueN, clientM)
	s.Start()

	return s.Server, nil
}
