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
	"github.com/dgrijalva/jwt-go"
	"fmt"
	"encoding/base64"
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
		signalTimer: 	time.NewTimer(time.Millisecond * 50),
		lock:        	&sync.Mutex{},
		heartBeatTime:	time.Now().UnixNano() / 1e6,
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
	if c.signalTimer != nil {
		c.signalTimer.Stop()
	}
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
			log.Trace("100000 %d", time.Now().UnixNano() / 1000000 - start)
			var m *pconnector.ToClientMessage
			func() {
				c.lock.Lock()
				defer c.lock.Unlock()
				log.Trace("200000 %d", time.Now().UnixNano() / 1000000 - start)
				for ;c.queue.Len() > 100; {
					if e := c.queue.Front(); e != nil {
						c.queue.Remove(e)
					}
				}
				log.Trace("300000 %d", time.Now().UnixNano() / 1000000 - start)
				if e := c.queue.Front(); e != nil {
					m = e.Value.(*pconnector.ToClientMessage)
					c.queue.Remove(e)
				}
				log.Trace("400000 %d", time.Now().UnixNano() / 1000000 - start)
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

			if time.Now().UnixNano() / 1000000 - m.Time > 2000 {
				log.Debug("discard2: %s, %s", m.MessageId, m.TimeText)
				continue
			}
			log.Trace("700000 %d", time.Now().UnixNano() / 1000000 - start)
			bytes, err := m.Marshal()
			if err == nil {
				c.msc.Output(pconnector.MsgTypePush, bytes)
				log.Trace("hehehehehe, payload=%s", m.TimeText)
			} else {
				log.Error("err: %s\r\n%s", err, debug.Stack())
			}
			log.Trace("800000 %d", time.Now().UnixNano() / 1000000 - start)
			timer := time.NewTimer(time.Millisecond * 50)
			<- timer.C
			log.Trace("900000 %d", time.Now().UnixNano() / 1000000 - start)
			timer.Stop()
			log.Trace("1000000 %d", time.Now().UnixNano() / 1000000 - start)
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
	claims, err0 := refreshToken0(register.Token, id)

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

	duration, err2 := c.sessionAliveDurationInSeconds(claims)
	if err2 != nil {
		return nil, err2
	}

	go c.expire(duration)

	return c, nil
}

func (c *Client)refreshToken(token string)  error {
	claims, err := refreshToken0(token, c.ID)
	if err != nil {
		return err
	}
	duration, err2 := c.sessionAliveDurationInSeconds(claims)
	if err2 != nil {
		return  err2
	}
	go c.expire(duration)

	return nil
}

func (c *Client)heartBeat() {
	timerLog.Info("handleHeartBeat, id=%s", c.ID)
	c.heartBeatTime = time.Now().UnixNano() / 1e6
}

func refreshToken0(token string, userId string) (jwt.MapClaims, error) {
	var claims jwt.MapClaims
	if config.Config.AuthKey != "" {
		decodeBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			return claims, err
		}
		token, err2 := jwt.Parse(fmt.Sprintf("%s", decodeBytes), func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(config.Config.AuthKey), nil
		})
		if err2 != nil {
			return claims, err2
		}

		var ok bool
		claims, ok = token.Claims.(jwt.MapClaims);
		if ok && token.Valid {
			audi := claims["aud"].(string)
			if audi != userId {
				return claims, fmt.Errorf("id not match(audi:%s, id:%s)", audi, userId)
			}
		} else {
			return claims, err2
		}
	}

	return claims, nil
}

func (c *Client) sessionAliveDurationInSeconds(claims jwt.MapClaims) (int64, error) {
	exp, ok := claims["exp"]
	if ok {
		now := time.Now().Unix()
		value, err := c.intValue(exp)
		if err != nil {
			return value, err
		}
		diff := value - now
		return diff, nil
	} else if config.Config.AuthKey == "" {
		return -1, nil
	}
	return 0, fmt.Errorf("cannot find exp in claims")
	/*
	iat, ok := claims["iat"]
	if ok {
		value, err := c.intValue(iat)
		return value, err
	}
	return int64(60) * 60 * 1000, nil*/
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

func (c *Client) expire(sessionAliveDurationInSeconds int64) {
	if sessionAliveDurationInSeconds > 0 {
		duration := time.Duration(sessionAliveDurationInSeconds) * time.Second
		if c.expireTimer == nil {
			c.expireTimer = time.NewTimer(duration)
			<- c.expireTimer.C
			log.Error("session to %s expired", c.ID)
			c.msc.Stop()
		}
		if c.expireTimer != nil {
			c.expireTimer.Reset(duration)
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
