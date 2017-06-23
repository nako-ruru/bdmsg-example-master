// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/someonegg/bdmsg"
	"golang.org/x/net/context"
	"net"
	"time"

	. "common/errdef"
	. "protodef/pconnector"
)

type client struct {
	*bdmsg.Client
	connected bool
}

func newClient(conn net.Conn, pumperInN, pumperOutN int) *client {

	c := &client{}

	mux := bdmsg.NewPumpMux(nil)

	c.Client = bdmsg.NewClient(nil, conn, bdmsg.DefaultIOC,
		mux, pumperInN, pumperOutN)
	c.doRegister()
	return c
}

func (c *client) doRegister() {
	var request Register
	// init request
	mr, _ := request.Marshal() // marshal request

	c.Client.Output(MsgTypeRegister, mr)
}

func (c *client) handleConnectReply(ctx context.Context,
	p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {

	var reply ConnectReply
	reply.Unmarshal(m) // unmarshal reply

	// process connect reply

	c.connected = true
}

func (c *client) handleServerHello(ctx context.Context,
	p *bdmsg.Pumper, t bdmsg.MsgType, m bdmsg.Msg) {

	var hello ServerHello
	err := hello.Unmarshal(m) // unmarshal hello
	if err != nil {
		panic(ErrParameter)
	}

	fmt.Println(hello.Message)
}

func (c *client) clientHello(msg string) {
	var hello ClientHello
	hello.Message = msg
	mr, _ := hello.Marshal()
	c.Output(MsgTypeClientHello, mr)
}

var serverAddr string

func parseFlag() {
	const (
		defaultAddr = "127.0.0.1:9001"
		usage       = "the server address"
	)
	flag.StringVar(&serverAddr, "addr", defaultAddr, usage)
	flag.StringVar(&serverAddr, "a", defaultAddr, usage+" (shorthand)")
	flag.Parse()
}

func main() {
	parseFlag()

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	c := newClient(conn, 100, 100)

	for {
		select {
		case <-time.After(time.Second):
		}
		if c.connected {
			fmt.Println("connected")
			break
		}
	}

	for q := false; !q; {
		select {
		case <-c.StopD():
			q = true
		case now := <-time.After(5 * time.Second):
			c.clientHello(fmt.Sprint(now))
		}
	}

	fmt.Println("disconnected")
}
