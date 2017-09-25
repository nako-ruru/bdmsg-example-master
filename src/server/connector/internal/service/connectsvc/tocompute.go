package connectsvc

import (
	"sync"
	"container/list"
)

type MessageQueueGroup struct {
	group map[string]*MessageQueue
	locker sync.RWMutex
}

type MessageQueue struct {
	queue list.List
	locker  sync.RWMutex
	size int
}

func (mqg *MessageQueueGroup)Add(msg FromConnectorMessage)  {
	mqg.locker.Lock()
	defer mqg.locker.Unlock()

	if mqg.group ==  nil {
		mqg.group = make(map[string]*MessageQueue)
	}
	if mqg.group[msg.RoomId] == nil {
		mqg.group[msg.RoomId] = NewMessageQueue()
	}
	mqg.group[msg.RoomId].Add(msg)
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

	return msgs, totalRestSize
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		size: 	100,
	}
}

func (m *MessageQueue) Add(msg FromConnectorMessage) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.queue.PushBack(msg)
	for ; m.queue.Len() > m.size; {
		for e := m.queue.Front(); e != nil; e = e.Next() {
			m.queue.Remove(e)
			break
		}
	}
}

func (m *MessageQueue) DrainTo(msgs []*FromConnectorMessage, maxLength int)([]*FromConnectorMessage, int)  {
	m.locker.Lock()
	defer m.locker.Unlock()

	oldLen := len(msgs)

	for e := m.queue.Front(); e != nil; e = e.Next() {
		if len(msgs) < maxLength {
			message := e.Value.(FromConnectorMessage)
			msgs = append(msgs, &message)
		} else {
			break
		}
	}
	for i,n := oldLen, len(msgs); i < n; i++ {
		for e := m.queue.Front(); e != nil; e = e.Next() {
			m.queue.Remove(e)
			break
		}
	}

	return msgs, m.queue.Len()
}