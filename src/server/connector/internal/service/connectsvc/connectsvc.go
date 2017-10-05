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
	"fmt"
	"strconv"
	"server/connector/internal/config"
	"github.com/Shopify/sarama"
	"github.com/bsm/sarama-cluster"
	"github.com/satori/go.uuid"
)

type service struct {
	*bdmsg.Server
	clientM *ClientManager
	roomM *RoomManager
}

var chatMessageCounter uint64 = 0
var packedMessageCounter uint64 = 0

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

func newService(l net.Listener, handshakeTO time.Duration, pumperInN, pumperOutN int, clientM *ClientManager, roomM *RoomManager) *service {

	s := &service{clientM: clientM, roomM: roomM}

	go subscribe(s)

	mux := bdmsg.NewPumpMux(nil)
	mux.HandleFunc(MsgTypeRegister, s.handleRegister)
	mux.HandleFunc(MsgTypeEnterRoom, s.handleEnterRoom)
	mux.HandleFunc(MsgTypeChat, s.handleMsg)

	s.Server = bdmsg.NewServerF(l, bdmsg.DefaultIOC, handshakeTO, mux, pumperInN, pumperOutN)

	initMessageConsumer()

	initRpcServerDiscovery()

	return s
}

//订阅
func subscribe(s *service) {
	groupID := uuid.NewV4().String()
	kConfig := cluster.NewConfig()
	kConfig.Consumer.Return.Errors = true
	kConfig.Group.Return.Notifications = true
	kConfig.Consumer.Offsets.CommitInterval = 1 * time.Second
	kConfig.Consumer.Offsets.Initial = sarama.OffsetNewest //初始从最新的offset开始

	c, err := cluster.NewConsumer(config.Config.Mq.KafkaBrokers, groupID, []string{config.Config.Mq.Topic}, kConfig)
	if err != nil {
		log.Error("Failed open consumer: %v", err)
		return
	}

	//defer c.Close()

	go func() {
		for err := range c.Errors() {
			log.Error("Error: %s\n", err.Error())
		}
	}()

	go func() {
		for note := range c.Notifications() {
			log.Info("Rebalanced: %+v\n", note)
		}
	}()

	for msg := range c.Messages() {
		log.Info("%s/%d/%d\t%s\n", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		c.MarkOffset(msg, "") //MarkOffset 并不是实时写入kafka，有可能在程序crash时丢掉未提交的offset
		handleSubscription(fmt.Sprintf("%s", msg.Value), s)
	}
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

	messageQueueGroup.Add(fromConnectorMessage)
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
			log.Error("err, %s", err) // 这里的err其实就是panic传入的内容，55
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
	deliver(readyToDeliver, restCount, i, start)
}
