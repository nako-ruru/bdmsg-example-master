// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connectsvc

import (
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
	"server/connector/internal/config"
	"fmt"
	"encoding/json"
)

type Client struct {
	ID        		string
	roomId    		string
	level       	int
	msc       		*bdmsg.SClient
	clientM   		*ClientManager
	room      		*RoomManager

	queue       	*list.List
	signalTimer 	*time.Timer
	expireTimer 	*time.Timer
	lock        	*sync.Mutex
	condition   	*sync.Cond
	heartBeatTime 	int64

	q         		bool

	version 		int
}

func createClient(id string, msc *bdmsg.SClient, clientM *ClientManager, room *RoomManager) (*Client, error) {
	t := &Client{
		ID:          	id,
		msc:         	msc,
		clientM:     	clientM,
		room:        	room,
		queue:       	list.New(),
		lock:        	&sync.Mutex{},
	}
	atomic.StoreInt64(&t.heartBeatTime, time.Now().UnixNano() / 1e6)
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

	e2 := c.msc.Pumper.Err()
	if e2 != nil {
		log.Error("%s", e2)
	}

	if e := recover(); e != nil {
		c.Close()

		const size = 16 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Error("Client$monitor, err=%s\r\n%s", e, fmt.Sprintf("\n%s", buf))
	}

	defer func() { recover() }()

	c.msc.SetUserData(nil)
	c.room.ending(c.roomId, c)
	c.clientM.removeClient(c)
	c.q = true
	if c.expireTimer != nil {
		c.expireTimer.Stop()
	}
	c.condition.Signal()
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
			start := time.Now().UnixNano() / 1000000

			/*
			for q := false; !q; {
				now := time.Now().UnixNano() / 1000000
				func() {
					c.lock.Lock()
					defer c.lock.Unlock()
					if e := c.queue.Front(); e != nil {
						message := e.Value.(*pconnector.ToClientMessage)
						if now - message.Time > 2000 {
							c.queue.Remove(e)
							return
						}
					}
					q = true
				}()
			}

			for q := false; !q; {
				statis := c.msc.Statis()
				func() {
					c.lock.Lock()
					defer c.lock.Unlock()
					size := config.Config.ServiceS.Connect.OutqueueN - int(statis.OutTotal - statis.OutProcess)
					if c.queue.Len() > size {
						if e := c.queue.Front(); e != nil {
							c.queue.Remove(e)
							return
						}
					}
					q = true
				}()
			}
			*/

			var m *pconnector.ToClientMessage
			func () {
				c.lock.Lock()
				defer c.lock.Unlock()
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
				log.Trace("500000 %d", time.Now().UnixNano() / 1000000 - start)
				continue
			}
			log.Trace("600000 %d", time.Now().UnixNano() / 1000000 - start)

			func() {
				log.Trace("700000 %d", time.Now().UnixNano() / 1000000 - start)
				bytes, err := m.Marshal()
				if err == nil {
					c.msc.Output(pconnector.MsgTypePush, bytes)
					log.Trace("800000 %d", time.Now().UnixNano() / 1000000 - start)
				} else {
					log.Error("err: %s\r\n%s", err, debug.Stack())
				}
			}()
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

func (m *ClientManager) clientIn(register pconnector.Register, msc *bdmsg.SClient, room *RoomManager) (*Client, error) {
	id := register.UserId
	claims, err0 := refreshToken0(register.Token)

	if err0 != nil {
		return nil, err0
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	c := m.clients[id]

	if c != nil {
		//return c, ErrAlreadyExist
		log.Info("anothor connection with same id(%s) comes in", id)
		c.msc.Stop()
	}

	c, err := createClient(id, msc, m, room)
	if err != nil {
		return nil, err
	}
	c.version = register.ClientToConnectorVersion

	m.clients[id] = c

	go c.monitor()
	go c.expire(claims.ExpireTime)

	return c, nil
}

func (c *Client)refreshToken(token string) error {
	claims, err := refreshToken0(token)
	if err != nil {
		return err
	}

	go c.expire(claims.ExpireTime)

	return nil
}

func (c *Client)heartBeat() {
	timerLog.Info("handleHeartBeat, id=%s", c.ID)
	atomic.StoreInt64(&c.heartBeatTime, time.Now().UnixNano() / 1e6)
}

func (c *Client)heartBeat2(timeTag int64) {
	now := time.Now().UnixNano() / 1e6
	atomic.StoreInt64(&c.heartBeatTime, now)
	timerLog.Info("handleHeartBeat, id=%s, clientTime=%s", c.ID, timeFormat(timeTag, "15:04:05.999"))
	if now - timeTag > 3000 {
		statis := c.msc.Statis()
		log.Error(
				"handleHeartBeat, id=%s, clientTime=%s, inQueue=%d",
				c.ID,
				timeFormat(timeTag, "15:04:05.999"),
				statis.InTotal - statis.InProcess,
		)
	}
}

func refreshToken0(tokenText string) (Token, error) {
	var token Token
	token.ExpireTime = -1
	if config.Config.PemFile != "" {
		return decrypt(tokenText)
	}
	return token, nil
}

func (c *Client) sessionAliveDurationInSeconds(claims Token) (int64, error) {
	if claims.ExpireTime > 0 {
		now := time.Now().Unix() / 1e6
		return claims.ExpireTime - now, nil
	}
	return -1, nil
}

func (c *Client)intValue(o interface{}) (int64, error)  {
	switch iat := o.(type) {
	case float64:
		return int64(iat), nil
	case json.Number:
		v, err := iat.Int64()
		return v, err
	}
	return 0, fmt.Errorf("cannot convert %v to int64", o)
}

func (c *Client) expire(expireTime int64) {
	if expireTime > 0 {
		now := time.Now().Unix() / 1e6
		durationInMills := expireTime - now
		if durationInMills > 0 {
			duration := time.Duration(durationInMills) * time.Millisecond
			if c.expireTimer == nil {
				c.expireTimer = time.AfterFunc(duration, func() {
					log.Error("session to %s expired", c.ID)
					c.msc.Stop()
				})
			}
			if c.expireTimer != nil {
				c.expireTimer.Reset(duration)
			}
		}
	}
}

func (m *ClientManager) CloseAll() {
	m.locker.RLock()
	defer m.locker.RUnlock()
	for _, c := range m.clients {
		c.Close()
	}
}

func (m *ClientManager) removeClient(c *Client) {
	m.locker.Lock()
	defer m.locker.Unlock()
	if m.clients[c.ID] == c {
		delete(m.clients, c.ID)
	}
}
