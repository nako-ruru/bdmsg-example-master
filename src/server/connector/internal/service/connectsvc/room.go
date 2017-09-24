package connectsvc

import (
	"server/connector/internal/manager"
	"sync"
)

type RoomManager struct {
	mSet *manager.ManagerSet

	locker  sync.RWMutex
	clients map[string]map[string]bool
}


func NewRoomManager(mSet *manager.ManagerSet) *RoomManager {
	return &RoomManager{
		mSet:    mSet,
		clients: make(map[string]map[string]bool),
	}
}

func (m *RoomManager) clientIn(id, roomId string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if roomId != "" {
		if m.clients[roomId] == nil {
			m.clients[roomId] = make(map[string]bool)
		}

		found := search(m, roomId, id)
		if !found {
			m.clients[roomId][id] = true
		}
	}
}

func search(m *RoomManager, roomId string, id string) bool {
	_, ok := m.clients[roomId][id]
	return ok
}

func (c *RoomManager) ending(roomId string, id string) {
	c.locker.Lock()
	defer c.locker.Unlock()

	delete(c.clients[roomId], id)
	if len(c.clients[roomId]) == 0 {
		delete(c.clients, roomId)
	}
}