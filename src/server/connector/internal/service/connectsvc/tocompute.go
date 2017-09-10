package connectsvc

import (
	"sync"
	"github.com/emirpasic/gods/lists/doublylinkedlist"
)

type RoomMsgToCompute struct {
	queue *doublylinkedlist.List
	locker  sync.RWMutex
	size int
}

type Entry struct {
	roomId string
	bytes []byte
}

func NewRoomMsgToCompute() *RoomMsgToCompute {
	return &RoomMsgToCompute{
		queue:    doublylinkedlist.New(),
		size: 1000,
	}
}


func (m *RoomMsgToCompute) Add(msg []byte) {
	m.queue.Add(msg)
	for ; m.queue.Size() > m.size; {
		log.Info("discard")
		m.queue.Remove(0)
	}
}

func (m *RoomMsgToCompute) DrainTo(roomId string, list *doublylinkedlist.List)()  {
	m.queue.Each(func(index int, value interface{}) {
		entry := Entry{
			roomId: roomId,
			bytes: value.([]byte),
		}
		list.Add(entry)
	})
}