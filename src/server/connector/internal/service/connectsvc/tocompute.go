package connectsvc

import (
	"sync"
	"container/list"
	"encoding/json"
)

type RoomMsgToCompute struct {
	queue list.List
	locker  sync.RWMutex
	size int
}

func NewRoomMsgToCompute() *RoomMsgToCompute {
	return &RoomMsgToCompute{
		size: 100,
	}
}


func (m *RoomMsgToCompute) Add(msg ToComputeMessage) {
	m.locker.Lock()
	defer m.locker.Unlock()

	m.queue.PushBack(msg)
	for ; m.queue.Len() > m.size; {
		for e := m.queue.Front(); e != nil; e = e.Next() {
			first := e.Value.(ToComputeMessage)
			jsonText, _ := json.Marshal(first)
			log.Info("discard: %s", jsonText)
			m.queue.Remove(e)
			break
		}
	}
}

func (m *RoomMsgToCompute) DrainTo(roomId string, list *list.List)  {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.queue.Len() <= 100000 {
		list.PushBackList(&m.queue)
		m.queue.Init()
	} else {
		count := 0;
		for e := m.queue.Front(); e != nil; e = e.Next() {
			list.PushBack(e.Value)
			m.queue.Remove(e)
			count ++
			if count >= 100000 {
				break;
			}
		}
	}
}