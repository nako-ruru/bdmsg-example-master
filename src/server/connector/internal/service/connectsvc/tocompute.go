package connectsvc

import (
	"sync"
	"container/list"
	"encoding/json"
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

func (mqg *MessageQueueGroup)Add(msg FromConnectorMessage)  int {
	mqg.locker.Lock()
	defer mqg.locker.Unlock()

	if mqg.group ==  nil {
		mqg.group = make(map[string]*MessageQueue)
	}
	if mqg.group[msg.RoomId] == nil {
		mqg.group[msg.RoomId] = NewMessageQueue()
	}
	mqg.group[msg.RoomId].Add(msg)

	totalRestSize := 0
	for _, v := range mqg.group {
		totalRestSize += v.queue.Len()
	}
	return totalRestSize
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
		if e := m.queue.Front(); e != nil {
			m.queue.Remove(e)
		}
	}

	return msgs, m.queue.Len()
}