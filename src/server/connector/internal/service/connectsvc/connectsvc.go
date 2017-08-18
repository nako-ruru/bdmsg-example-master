// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connectsvc

import (
	"github.com/someonegg/bdmsg"
	"github.com/someonegg/golog"
	"golang.org/x/net/context"
	"net"
	"time"

	. "common/config"
	. "common/errdef"
	. "protodef/pconnector"
	"encoding/json"
	"github.com/go-redis/redis"
	"fmt"
)

var log = golog.SubLoggerWithFields(golog.RootLogger, "module", "connectsvc")

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:9921",
	Password: "BrightHe0", // no password set
	DB:       0,  // use default DB
})

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
			fmt.Println("Receive from channel:", msg.Channel, msg.Payload)
			if err != nil {
				break
			}
			var hello PushMsg
			hello.Unmarshal([]byte(msg.Payload))
			i, ok := s.clientM.clients[hello.UserId]
			fmt.Println("ok1:", ok)
			if ok {
				fmt.Println("ok2:", ok)
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
		log.Error("handleRegister", "error", err)
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


	log.Info("handleMsg", "bdmsg.Msg", fmt.Sprintf("%v : %v : %v", c.ID, t, string(m[:])))

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

	var redisMsg = RedisMsg{roomId, c.ID, time.Now().UnixNano() / 1000000, int(t), nickname, level, params}
	var bytes,_ = json.Marshal(redisMsg)

	pong, err := client.Ping().Result()
	if err != nil {
		fmt.Println(pong, err)
	}


	var sendDirectly = false
	if(sendDirectly) {
		client.RPush(roomId, bytes)
	} else {
		client.Publish("connector", string(bytes))
	}
}

func Start(conf *BDMsgSvcConfT, clientM *ClientManager) (*bdmsg.Server, error) {
	l, err := net.ListenTCP("tcp", (*net.TCPAddr)(&conf.ListenAddr))
	if err != nil {
		log.Error("Start$net.ListenTCP", "error", err)
		return nil, ErrAddress
	}

	s := newService(l, time.Duration(conf.HandshakeTO), conf.InqueueN, conf.OutqueueN, clientM)
	s.Start()

	return s.Server, nil
}
