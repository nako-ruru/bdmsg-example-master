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

type service struct {
	*bdmsg.Server
	clientM *ClientManager
}

func newService(l net.Listener, handshakeTO time.Duration, pumperInN, pumperOutN int,
	clientM *ClientManager) *service {

	s := &service{clientM: clientM}

	mux := bdmsg.NewPumpMux(nil)
	mux.HandleFunc(MsgTypeRegister, s.handleConnect)
	mux.HandleFunc(MsgTypeEnterRoom, s.handleChat)
	mux.HandleFunc(MsgTypeChat, s.handleChat)
	mux.HandleFunc(MsgTypeSupport, s.handleChat)
	mux.HandleFunc(MsgTypeSendGift, s.handleChat)
	mux.HandleFunc(MsgTypeShare, s.handleChat)
	mux.HandleFunc(MsgTypeLevelUp, s.handleChat)

	s.Server = bdmsg.NewServerF(l, bdmsg.DefaultIOC, handshakeTO,
		mux, pumperInN, pumperOutN)
	return s
}

func (s *service) handleConnect(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
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
		log.Error("handleConnect", "error", err)
		panic(ErrUnexpected)
	}

	// tell bdmsg that client is authorized
	msc.Handshake()
}

func (s *service) handleChat(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)

	var roomId string
	var level int
	var nickname string
	var params map[string]string;

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
		err := hello.Unmarshal(m) // unmarshal hello
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
		err := hello.Unmarshal(m) // unmarshal hello
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
		err := hello.Unmarshal(m) // unmarshal hello
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
		err := hello.Unmarshal(m) // unmarshal hello
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

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := client.Ping().Result()
	fmt.Println(pong, err)

	client.RPush(roomId, bytes)
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
