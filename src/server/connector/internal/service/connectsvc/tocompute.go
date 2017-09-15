package connectsvc

import (
	"sync"
	"github.com/emirpasic/gods/lists/singlylinkedlist"
)

type RoomMsgToCompute struct {
	queue *singlylinkedlist.List
	locker  sync.RWMutex
	size int
}

func NewRoomMsgToCompute() *RoomMsgToCompute {
	return &RoomMsgToCompute{
		queue:	singlylinkedlist.New(),
		size: 	100,
	}
}


func (m *RoomMsgToCompute) Add(msg ToComputeMessage) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.queue.Add(msg)
	for ; m.queue.Size() > m.size; {
		m.queue.Remove(0)
	}
}

func (m *RoomMsgToCompute) DrainTo(roomId string, msgs []ToComputeMessage, maxLength int)([]ToComputeMessage)  {
	m.locker.Lock()
	defer m.locker.Unlock()

	oldLen := len(msgs)

	m.queue.Each(func(index int, value interface{}) {
		if len(msgs) < maxLength {
			msgs = append(msgs, value.(ToComputeMessage))
		}
	})
	for i,n := oldLen, len(msgs); i < n; i++ {
		m.queue.Remove(0)
	}

	return msgs
}