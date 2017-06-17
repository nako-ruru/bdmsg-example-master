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
	mux.HandleFunc(MsgTypeConnect, s.handleConnect)
	mux.HandleFunc(MsgTypeClientHello, s.handleClientHello)

	s.Server = bdmsg.NewServerF(l, bdmsg.DefaultIOC, handshakeTO,
		mux, pumperInN, pumperOutN)
	return s
}

func (s *service) handleConnect(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	msc := p.UserData().(*bdmsg.SClient)
	if msc.Handshaked() {
		panic(ErrUnexpected)
	}

	var request ConnectRequst
	err := request.Unmarshal(m) // unmarshal request
	if err != nil {
		panic(ErrParameter)
	}

	_, err = s.clientM.clientIn(request.ID, request.Pass, msc)
	if err != nil {
		log.Error("handleConnect", "error", err)
		panic(ErrUnexpected)
	}

	// tell bdmsg that client is authorized
	msc.Handshake()

	var reply ConnectReply
	// init reply
	mr, _ := reply.Marshal() // marshal reply
	msc.Output(MsgTypeConnectReply, mr)
}

func (s *service) handleClientHello(ctx context.Context, p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {
	c := p.UserData().(*Client)

	var hello ClientHello
	err := hello.Unmarshal(m) // unmarshal hello
	if err != nil {
		panic(ErrParameter)
	}

	c.ClientHello(hello.Message)
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
