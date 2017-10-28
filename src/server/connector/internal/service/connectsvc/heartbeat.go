package connectsvc

import (
	"time"
	"server/connector/internal/config"
)

func initHeartBeat(service *service)  {
	if !config.Config.DisableHeartBeat {
		service.heartBeatTimer = time.NewTimer(time.Second * 1)
		for  {
			<- service.heartBeatTimer.C
			now := time.Now().UnixNano() / 1e6
			from := now - 3000
			func() {
				service.clientM.locker.RLock()
				defer service.clientM.locker.RUnlock()

				for _, client := range service.clientM.clients {
					if client.version >= 100  && client.heartBeatTime < from {
						log.Info("heart beat time out, id=%s", client.ID)
						client.Close()
					}
				}
			}()
			service.heartBeatTimer.Reset(time.Second * 1)
		}
	}
}