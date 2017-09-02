package connectsvc

import (
	"server/connector/internal/manager"
	"sync"
)

type RoomManager struct {
	mSet *manager.ManagerSet

	locker  sync.RWMutex
	clients map[string][]string
}


func NewRoomManager(mSet *manager.ManagerSet) *RoomManager {
	return &RoomManager{
		mSet:    mSet,
		clients: make(map[string][]string),
	}
}

func (m *RoomManager) clientIn(id, roomId string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.clients[roomId] == nil {
		m.clients[roomId] = []string{}
	}

	m.internalEnding(id)

	m.clients[roomId] = append(m.clients[roomId], id)
}

func (c *RoomManager) ending(id string) {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.internalEnding(id)
}
func (c *RoomManager) internalEnding(id string) {
	for k, v := range c.clients {
		elements := []string{}
		for _, e := range v {
			if e == id {
				continue
			} else {
				elements = append(elements, e)
			}
		}
		c.clients[k] = elements
	}
}