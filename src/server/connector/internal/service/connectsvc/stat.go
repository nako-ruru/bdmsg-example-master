package connectsvc

import (
	"time"
	"server/connector/internal/config"
	"sync/atomic"
	"net"
)

type NamingInfo struct {
	RegisterTime     int64 		`json:"registerTime"`
	LoginUsers       int32   	`json:"loginUsers"`
	ConnectedClients int32   	`json:"connectedClients"`
	InData			 int64 		`json:"inData"`
	OutData			 int64 		`json:"outData"`
	InQueue			 int32   	`json:"inQueue"`
	OutQueue	     int32 		`json:"outQueue"`
}

var info = NamingInfo{}

func RegisterNamingService(service *service, clientManager *ClientManager)  {
	log.Debug("register %s to %s periodically", getInternetAddress(), config.Config.Redis.Addresses)

	var f = func() {
		var client = newNamingRedisClient()
		defer client.Close()

		clientManager.locker.Lock()
		defer clientManager.locker.Unlock()

		tempInfo := NamingInfo{
			RegisterTime : 		time.Now().UnixNano() / 1000000,
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

type listener struct {
	net.Listener
}
type myConn struct {
	net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		atomic.AddInt32(&info.ConnectedClients, 1)
	}
	return NewConn(c), err
}

func (c *myConn) Close() error {
	err := c.Conn.Close()
	if err == nil {
		atomic.AddInt32(&info.ConnectedClients, -1)
	}
	return err
}

func NewListener(inner net.Listener) *listener {
	l := new(listener)
	l.Listener = inner
	return l
}

func NewConn(inner net.Conn) *myConn {
	conn := new(myConn)
	conn.Conn = inner
	return conn
}