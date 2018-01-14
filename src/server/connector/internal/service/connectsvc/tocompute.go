package connectsvc

import (
	"sync"
	"container/list"
	"encoding/json"
	"time"
)

type MessageQueueGroup struct {
	group map[string]*MessageQueue
	locker sync.RWMutex
}

type MessageQueue struct {
	queue   list.List
	locker  sync.RWMutex
	maxSize int
}

func (mqg *MessageQueueGroup)Add(msg FromConnectorMessage) {
	var mq *MessageQueue

	start := time.Now().UnixNano() / 1000000
	log.Trace("100000 %d", time.Now().UnixNano() / 1000000 - start)
	func() {
		mqg.locker.RLock()
		defer mqg.locker.RUnlock()
		if mqg.group != nil {
			mq = mqg.group[msg.RoomId]
		}
	}()

	log.Trace("200000 %d", time.Now().UnixNano() / 1000000 - start)
	if mq == nil {
		func() {
			mqg.locker.Lock()
			defer mqg.locker.Unlock()
			if mqg.group ==  nil {
				mqg.group = make(map[string]*MessageQueue)
			}
			if mqg.group[msg.RoomId] == nil {
				mqg.group[msg.RoomId] = NewMessageQueue()
			}
			mq = mqg.group[msg.RoomId]
		}()
	}

	log.Trace("300000 %d", time.Now().UnixNano() / 1000000 - start)
	mq.Add(msg)

	log.Trace("400000 %d", time.Now().UnixNano() / 1000000 - start)
}

func (mqg *MessageQueueGroup) DrainTo(msgs []*FromConnectorMessage, maxLength int) ([]*FromConnectorMessage, int)  {
	mqg.locker.RLock()
	defer mqg.locker.RUnlock()
	totalRestSize := 0
	for _, v := range mqg.group {
		var restSize int
		msgs, restSize = v.DrainTo(msgs, maxLength)
		totalRestSize += restSize
		log.Debug("size: %d", len(msgs))
	}

	log.Trace("60000000000")
	return msgs, totalRestSize
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		maxSize: 100,
	}
}

func (m *MessageQueue) Add(msg FromConnectorMessage) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.queue.PushBack(msg)
	for ; m.queue.Len() > m.maxSize; {
		if e := m.queue.Front(); e != nil {
			m.queue.Remove(e)
			var jsonText, _ = json.Marshal(e.Value.(FromConnectorMessage))
			log.Warn("discard: %s", jsonText)
		}
	}
}

func (m *MessageQueue) DrainTo(msgs []*FromConnectorMessage, maxLength int)([]*FromConnectorMessage, int)  {
	log.Trace("10000000000")
	m.locker.Lock()
	defer m.locker.Unlock()

	log.Trace("20000000000")
	oldLen := len(msgs)

	log.Trace("30000000000")
	for e := m.queue.Front(); e != nil; e = e.Next() {
		if len(msgs) < maxLength {
			message := e.Value.(FromConnectorMessage)
			msgs = append(msgs, &message)
		} else {
			break
		}
	}
	log.Trace("40000000000")
	for i,n := oldLen, len(msgs); i < n; i++ {
		if e := m.queue.Front(); e != nil {
			m.queue.Remove(e)
		}
	}
	log.Trace("50000000000")

	return msgs, m.queue.Len()
}