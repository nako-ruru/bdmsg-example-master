package connectsvc

import (
	"server/connector/internal/manager"
	"sync"
	"github.com/emirpasic/gods/sets/treeset"
)

type RoomManager struct {
	mSet *manager.ManagerSet

	locker  sync.RWMutex
	clients map[string]*treeset.Set
}

type Room struct {
	 userIds *treeset.Set
	locker  sync.RWMutex
}

func NewRoomManager(mSet *manager.ManagerSet) *RoomManager {
	return &RoomManager{
		mSet:    mSet,
		clients: make(map[string]*treeset.Set),
	}
}

func (m *RoomManager) clientIn(id, roomId string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if roomId != "" {
		if m.clients[roomId] == nil {
			m.clients[roomId] = treeset.NewWithStringComparator()
		}

		found := search(m, roomId, id)
		if !found {
			m.clients[roomId].Add(id)
		}
	}
}

func search(m *RoomManager, roomId string, id string) bool {
	return m.clients[roomId].Contains(id)
}

func (c *RoomManager) ending(id string) {
	c.locker.Lock()
	defer c.locker.Unlock()

	c.internalEnding(id)
}

func (c *RoomManager) internalEnding(id string) {
	for k := range c.clients {
		c.clients[k].Remove(id)
	}
}