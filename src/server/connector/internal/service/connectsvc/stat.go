package connectsvc

import "time"
import (
	"server/connector/internal/config"
	"sync/atomic"
)

type NamingInfo struct {
	RegisterTime     int64 `json:"registerTime"`
	LoginUsers       int   `json:"loginUsers"`
	ConnectedClients int   `json:"connectedClients"`
	InData			 int64 `json:"inData"`
	OutData			 int64 `json:"outData"`
	InQueue			 int32   `json:"inQueue"`
	OutQueue	     int32 `json:"outQueue"`
}

var info = NamingInfo{}

func RegisterNamingService(clientManager *ClientManager)  {
	log.Debug("register %s to %s periodically", getInternetAddress(), config.Config.Redis.Addresses)

	var f = func() {
		var client = newNamingRedisClient()
		defer client.Close()

		clientManager.locker.Lock()
		defer clientManager.locker.Unlock()

		tempInfo := NamingInfo{
			RegisterTime : 		info.RegisterTime,
			LoginUsers : 		info.LoginUsers,
			ConnectedClients :	info.ConnectedClients,
			InData	 :		 	info.InData,
			OutData	 :		 	info.OutData,
			InQueue	 :		 	info.InQueue,
			OutQueue :	     	info.OutQueue,
		}

		atomic.StoreInt64(&info.OutData, 0)
		atomic.StoreInt64(&info.InData, 0)

		jsonText, err := tempInfo.Marshal()
		if err == nil {
			client.HSet("go-servers", getInternetAddress(), jsonText)
		} else {

		}
	}

	ticker := time.NewTicker(time.Second * 1)
	go func() {
		for range ticker.C {
			f()
		}
	}()
}