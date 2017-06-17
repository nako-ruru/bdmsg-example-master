// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connectsvc

import (
	"fmt"
	"github.com/someonegg/bdmsg"
	"runtime"
	"sync"

	. "common/errdef"
	. "protodef/pconnector"
	"server/connector/internal/manager"
)

type Client struct {
	ID      string
	msc     *bdmsg.SClient
	clientM *ClientManager

	msgC chan string
}

func createClient(id, pass string, msc *bdmsg.SClient, clientM *ClientManager) (*Client, error) {
	log.Info("createClient", "id", id, "remoteaddr", msc.Conn().RemoteAddr())

	t := &Client{
		ID:      id,
		msc:     msc,
		clientM: clientM,
		msgC:    make(chan string, 1),
	}

	t.msc.SetUserData(t)
	return t, nil
}

func (c *Client) monitor() {
	defer c.ending()

	c.ServerHello("SERVER_INITED")

	for q := false; !q; {
		select {
		case <-c.msc.StopD():
			q = true
		case msg := <-c.msgC:
			log.Info("Client$monitor$clienthello", "content", msg)
			c.ServerHello("SERVER_RECEIVED " + msg)
		}
	}

	stat := c.msc.Statis()
	log.Info("Client$monitor$", "id", c.ID, "mscerror", c.msc.Err(), "mscstat", stat)
}

func (c *Client) ending() {
	if e := recover(); e != nil {
		c.Close()

		const size = 16 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Error("Client$monitor", "error", "panic", "stack", fmt.Sprintf("\n%s", buf))
	}

	defer func() { recover() }()

	c.msc.SetUserData(nil)
	c.clientM.removeClient(c.ID)
}

// Never fail.
func (c *Client) Close() error {
	c.msc.Stop()
	return nil
}

// Not nil.
func (c *Client) MSC() *bdmsg.SClient {
	return c.msc
}

func (c *Client) ClientHello(msg string) {
	select {
	case c.msgC <- msg:
	default:
		// discard
	}
}

func (c *Client) ServerHello(msg string) {
	var hello ServerHello
	hello.Message = msg
	mr, _ := hello.Marshal()
	c.msc.Output(MsgTypeServerHello, mr)
}

type ClientManager struct {
	mSet *manager.ManagerSet

	locker  sync.RWMutex
	clients map[string]*Client
}

func NewClientManager(mSet *manager.ManagerSet) *ClientManager {
	return &ClientManager{
		mSet:    mSet,
		clients: make(map[string]*Client),
	}
}

func (m *ClientManager) Client(id string) *Client {
	m.locker.RLock()
	defer m.locker.RUnlock()
	return m.clients[id]
}

func (m *ClientManager) clientIn(id, pass string, msc *bdmsg.SClient) (*Client, error) {
	m.locker.Lock()
	defer m.locker.Unlock()

	c := m.clients[id]
	if c != nil {
		return c, ErrAlreadyExist
	}

	c, err := createClient(id, pass, msc, m)
	if err != nil {
		return nil, err
	}

	m.clients[id] = c
	go c.monitor()

	return c, nil
}

func (m *ClientManager) CloseAll() {
	m.locker.RLock()
	defer m.locker.RUnlock()
	for _, c := range m.clients {
		c.Close()
	}
}

func (m *ClientManager) removeClient(id string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	delete(m.clients, id)
}
