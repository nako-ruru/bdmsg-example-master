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
	"sync/atomic"
	"strconv"
	"runtime/debug"
)

type service struct {
	*bdmsg.Server
	clientM *ClientManager
	roomM *RoomManager
	listener *listener
	heartBeatTimer *time.Timer
}

var chatMessageCounter uint64 = 0
var packedMessageCounter uint64 = 0

func Start(conf *BDMsgSvcConfT, clientM *ClientManager, roomM* RoomManager) (*bdmsg.Server, error) {
	l, err := net.ListenTCP("tcp", (*net.TCPAddr)(&conf.ListenAddr))
	if err != nil {
		log.Error("Start$net.ListenTCP, err=%s\r\n%s", err, debug.Stack())
		return nil, ErrAddress
	}

	s := newService(l, time.Duration(conf.HandshakeTO), conf.InqueueN, conf.OutqueueN, clientM, roomM)
	s.Start()

	return s.Server, nil
}

func newService(l net.Listener, handshakeTO time.Duration, pumperInN, pumperOutN int, clientM *ClientManager, roomM *RoomManager) *service {
	listener := NewListener(l)

	s := &service{clientM: clientM, roomM: roomM, listener: listener}

	go subscribe(s)

	mux := bdmsg.NewPumpMux(nil)
	mux.HandleFunc(MsgTypeRegister, s.handleRegister)
	mux.HandleFunc(MsgTypeEnterRoom, s.handleEnterRoom)
	mux.HandleFunc(MsgTypeChat, s.handleMsg)
	mux.HandleFunc(MsgTypeRefreshToken, s.handleRefreshToken)
	mux.HandleFunc(MsgTypeHeartBeat, s.handleHeartBeat)

	s.Server = bdmsg.NewServerF(listener, bdmsg.DefaultIOC, handshakeTO, mux, pumperInN, pumperOutN)

	RegisterNamingService(s)
	defer UnregisterNamingService()

	go initHeartBeat(s)

	initMessageConsumer()

	initRpcServerDiscovery()

	return s
}

/*
  客户端登记
 */
func (s *service) handleRegister(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	msc := p.UserData().(*bdmsg.SClient)

	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			log.Error("err, %s\r\n%s", err, debug.Stack()) // 这里的err其实就是panic传入的内容，55
			msc.Stop()
		}
	}()

	if msc.Handshaked() {
		panic(ErrUnexpected)
	}

	var register Register
	err := register.Unmarshal(m) // unmarshal register
	if err != nil {
		panic(ErrParameter)
	}

	_, err = s.clientM.clientIn(register, msc, s.roomM)
	if err != nil {
		log.Error("handleRegister, err=%s\r\n%s", err, debug.Stack())
		panic(ErrUnexpected)
	} else {
		log.Info("handleRegister, id=%s, version=%d, remoteaddr=%s", register.UserId, register.ClientToConnectorVersion, msc.Conn().RemoteAddr())
	}

	// tell bdmsg that client is authorized
	msc.Handshake()
}

func (s *service)handleRefreshToken(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)
	c.heartBeat()

	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			log.Error("err, %s\r\n%s", err, debug.Stack()) // 这里的err其实就是panic传入的内容，55
			c.msc.Stop()
		}
	}()

	var refreshToken RefreshToken
	err := refreshToken.Unmarshal(m) // unmarshal refreshToken
	if err != nil {
		panic(ErrParameter)
	}
	log.Info("handleRefreshToken, id=%s", c.ID)
	c.refreshToken(refreshToken.Token)
}


func (s *service)handleHeartBeat(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)
	c.heartBeat()
}

func (s *service) handleEnterRoom(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)
	c.heartBeat()

	var enterRoom EnterRoom
	err := enterRoom.Unmarshal(m) // unmarshal enterRoom
	if err != nil {
		panic(ErrParameter)
	}

	s.roomM.clientIn(c, enterRoom.RoomId)
	c.level = enterRoom.Level
	log.Info("handleEnterRoom, id=%s, roomId=%s", c.ID, enterRoom.RoomId)
}

func (s *service) handleMsg(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)
	c.heartBeat()

	var roomId string = c.roomId
	var level int
	var nickname string
	var params map[string]string;


	log.Info("handleMsg, id=%s, roomId=%s, t=%d, time=%d, m=%s", c.ID, roomId, t, time.Now().UnixNano() / 1000000, string(m[:]))

	switch t {
	case 1:
		var chat Chat
		err := chat.Unmarshal(m) // unmarshal chat
		if err != nil {
			panic(ErrParameter)
		}
		if chat.RoomId != "" && chat.RoomId != roomId {
			roomId = chat.RoomId
			s.roomM.clientIn(c, roomId)
		}
		level = chat.Level
		nickname = chat.Nickname
		params = map[string]string{"content": chat.Content}
		break
	}

	var fromConnectorMessage = FromConnectorMessage{
		strconv.FormatUint(atomic.AddUint64(&chatMessageCounter, 1), 10),
		roomId,
		c.ID,
		nickname,
		time.Now().UnixNano() / 1000000.,
		int32(level),
		int32(t),
		params,
	}

	totalSize := messageQueueGroup.Add(fromConnectorMessage)
	info.InQueue = int32(totalSize)
}

var messageQueueGroup MessageQueueGroup

func initMessageConsumer() {
	ticker := time.NewTicker(time.Millisecond * 50)
	go func() {
		for range ticker.C {
			consume(0)
		}
	}()
}

func consume(id int) {
	i := atomic.AddUint64(&packedMessageCounter, 1)
	consumeEvent(i, id)
}

func consumeEvent(i uint64, id int) {
	defer func(){ // 必须要先声明defer，否则不能捕获到panic异常
		if err:=recover();err!=nil{
			log.Error("err, %s\r\n%s", err, debug.Stack()) // 这里的err其实就是panic传入的内容，55
			print(err)
		}
	}()

	log.Debug("consume: <- produceEvt")
	readyToDeliver := []*FromConnectorMessage{}
	start := time.Now().UnixNano() / 1000000
	log.Debug("consume, id=%d, event=%d, time=%d", id, i, start)
	maxCount := 10000
	var restCount int
	readyToDeliver, restCount = messageQueueGroup.DrainTo(readyToDeliver, maxCount)
	rpcClient.deliver(readyToDeliver, restCount, i, start)
}
