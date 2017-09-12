package connectsvc

import (
	"sync"
	"github.com/emirpasic/gods/lists/doublylinkedlist"
	"encoding/json"
)

type RoomMsgToCompute struct {
	queue *doublylinkedlist.List
	locker  sync.RWMutex
	size int
}

func NewRoomMsgToCompute() *RoomMsgToCompute {
	return &RoomMsgToCompute{
		queue:    doublylinkedlist.New(),
		size: 100,
	}
}


func (m *RoomMsgToCompute) Add(msg ToComputeMessage) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.queue.Add(msg)
	for ; m.queue.Size() > m.size; {
		e, _ := m.queue.Get(0)
		c := e.(ToComputeMessage)
		text, _ := json.Marshal(c)
		log.Info("discard: %s", text)
		m.queue.Remove(0)
	}
}

func (m *RoomMsgToCompute) DrainTo(roomId string, list *doublylinkedlist.List)()  {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.queue.Each(func(index int, value interface{}) {
		if(index < 100000) {
			list.Add(value)
		}
	})
	for i,n := 0, list.Size(); i < n; i++ {
		m.queue.Remove(0)
	}
}

func min(a int, b int)(int) {
	if a <= b {
		return a;
	}
	return b;
}