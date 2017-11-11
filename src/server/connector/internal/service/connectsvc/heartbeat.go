package connectsvc

import (
	"time"
	"server/connector/internal/config"
	"sync/atomic"
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
					heartBeatTime := atomic.LoadInt64(&client.heartBeatTime)
					if client.version >= 100  && heartBeatTime < from {
						statis := client.msc.Statis()
						log.Error(
							"heart beat time out, id=%s, now=%s, client.heartBeatTime=%s, client.inQueue=%d",
								client.ID,
								timeFormat(now, "15:04:05.999"),
								timeFormat(heartBeatTime,"15:04:05.999"),
							    statis.InTotal - statis.InProcess,
						)
						client.Close()
					}
				}
			}()
			service.heartBeatTimer.Reset(time.Second * 1)
		}
	}
}