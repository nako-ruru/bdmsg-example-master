package connectsvc

import (
	"server/connector/internal/manager"
	"sync"
)

type RoomManager struct {
	locker  sync.RWMutex
	clients map[string]map[string]*Client
}


func NewRoomManager(mSet *manager.ManagerSet) *RoomManager {
	return &RoomManager{
		clients: make(map[string]map[string]*Client),
	}
}

func (m *RoomManager) clientIn(c *Client, roomId string) {
	if roomId == "" {
		log.Error("enter room failed: (id:%s, roomId:%s)", c.ID, roomId)
	} else {
		m.ending(roomId, c)

		m.locker.Lock()
		defer m.locker.Unlock()

		if roomId != "" {
			if m.clients[roomId] == nil {
				m.clients[roomId] = make(map[string]*Client)
			}

			found := search(m, roomId, c)
			if !found {
				m.clients[roomId][c.ID] = c
			}
			c.roomId = roomId
		}
	}
}

func search(m *RoomManager, roomId string, client *Client) bool {
	oldClient, _ := m.clients[roomId][client.ID]
	return oldClient == client
}

func (m *RoomManager) ending(roomId string, client *Client) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if m.clients[roomId][client.ID] == client {
		delete(m.clients[roomId], client.ID)
		if len(m.clients[roomId]) == 0 {
			delete(m.clients, roomId)
		}
	}
}