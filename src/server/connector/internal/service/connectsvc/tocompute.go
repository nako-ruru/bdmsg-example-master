package connectsvc

import (
	"sync"
	"container/list"
)

type RoomMsgToCompute struct {
	queue list.List
	locker  sync.RWMutex
	size int
}

func NewRoomMsgToCompute() *RoomMsgToCompute {
	return &RoomMsgToCompute{
		size: 	100,
	}
}


func (m *RoomMsgToCompute) Add(msg FromConnectorMessage) {
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

func (m *RoomMsgToCompute) DrainTo(roomId string, msgs []*FromConnectorMessage, maxLength int)([]*FromConnectorMessage, int)  {
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