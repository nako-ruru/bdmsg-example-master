// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connectsvc

import (
	"fmt"
	"github.com/someonegg/bdmsg"
	"runtime"
	"sync"

	_ "common/errdef"
	_ "protodef/pconnector"
	"server/connector/internal/manager"
	"protodef/pconnector"
	"container/list"
	"time"
	"runtime/debug"
	"sync/atomic"
	"bytes"
)

type Client struct {
	ID        	string
	roomId    	string
	level       int
	msc       	*bdmsg.SClient
	clientM   	*ClientManager
	room      	*RoomManager

	queue     *list.List
	timer     *time.Timer
	lock  		*sync.Mutex
	condition	*sync.Cond

	in        bytes.Buffer
	q         bool
}

func createClient(id, pass string, msc *bdmsg.SClient, clientM *ClientManager, room *RoomManager) (*Client, error) {
	t := &Client{
		ID:      id,
		msc:     msc,
		clientM: clientM,
		room:    room,
		queue:   list.New(),
		timer:   time.NewTimer(time.Millisecond * 50),
		lock:    &sync.Mutex{},
	}
	t.condition = sync.NewCond(t.lock)

	atomic.AddInt32(&info.LoginUsers, 1)

	go func() {
		t.a()
	}()

	t.msc.SetUserData(t)
	return t, nil
}

func (c *Client) monitor() {
	defer c.ending()

	for q := false; !q; {
		select {
		case <-c.msc.StopD():
			q = true
		}
	}
}

func (c *Client) ending() {
	log.Info("Client$ending, id=%s, roomId=%s", c.ID, c.roomId)

	if e := recover(); e != nil {
		c.Close()

		const size = 16 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Error("Client$monitor, buf=%s", fmt.Sprintf("\n%s", buf))
	}

	defer func() { recover() }()

	c.msc.SetUserData(nil)
	c.room.ending(c.roomId, c.ID)
	c.clientM.removeClient(c.ID)
	c.timer.Stop()
	c.q = true
	atomic.AddInt32(&info.LoginUsers, -1)
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

func (c *Client) ServerHello(hello *pconnector.ToClientMessage) {
	if c == nil {
		log.Warn("ServerHello, c == nil")
	} else if c.msc == nil {
		log.Warn("ServerHello, c.msc == nil")
	} else {
		c.lock.Lock()
		defer c.lock.Unlock()
		log.Trace("90000:%s", hello.TimeText)
		c.queue.PushBack(hello)
		c.condition.Broadcast()
	}
}

func (c *Client)a()  {
	if c == nil {
		log.Warn("ServerHello, c == nil")
	} else if c.msc == nil {
		log.Warn("ServerHello, c.msc == nil")
	} else {
		for ;!c.q; {
			var m *pconnector.ToClientMessage
			func() {
				c.lock.Lock()
				defer c.lock.Unlock()
				for ;c.queue.Len() > 100; {
					if e := c.queue.Front(); e != nil {
						c.queue.Remove(e)
					}
				}
				if e := c.queue.Front(); e != nil {
					m = e.Value.(*pconnector.ToClientMessage)
					c.queue.Remove(e)
				}
			}()
			if m == nil {
				func() {
					c.lock.Lock()
					defer c.lock.Unlock()
					c.condition.Wait()
				}()
				timer := time.NewTimer(time.Millisecond * 50)
				<- timer.C
				timer.Stop()
				continue
			}

			if time.Now().UnixNano() / 1000000 - m.Time > 2000 {
				log.Trace("discard2: %s, %s", m.MessageId, m.TimeText)
				continue
			}
			bytes, err := m.Marshal()
			if err == nil {
				c.msc.Output(pconnector.MsgTypePush, bytes)
				atomic.AddInt64(&info.OutData, int64(len(bytes)))
				log.Trace("hehehehehe, payload=%s", m.TimeText)
			} else {
				log.Error("%s", debug.Stack())
			}
		}
	}
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

func (m *ClientManager) clientIn(id, pass string, msc *bdmsg.SClient, room *RoomManager) (*Client, error) {
	m.locker.Lock()
	defer m.locker.Unlock()

	c := m.clients[id]
	if c != nil {
		//return c, ErrAlreadyExist
	}

	c, err := createClient(id, pass, msc, m, room)
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
