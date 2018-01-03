package connectsvc

import (
	"time"
	"server/connector/internal/config"
	"sync/atomic"
	"net"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"runtime/debug"
)

type NamingInfo struct {
	RegisterTime     int64   `json:"registerTime"`
	LoginUsers       int32   `json:"loginUsers"`
	ConnectedClients int32   `json:"connectedClients"`
	InData           int64   `json:"inData"`
	OutData          int64   `json:"outData"`
	InQueue          int32   `json:"inQueue"`
	OutQueue         int32   `json:"outQueue"`
	CPU              []float64   `json:"cpuUsage"`
	MemoryUsage      float64 `json:"memoryUsage""`
}

var info = NamingInfo{}

func RegisterNamingService(service *service)  {
	log.Info("register %s to %s periodically", getInternetAddress(), config.Config.Redis.Addresses)

	var f = func() {
		var client = newNamingRedisClient()
		defer client.Close()

		var totalIn, totalOut int64 = 0, 0
		func() {
			service.clientM.locker.RLock()
			defer service.clientM.locker.RUnlock()

			for _, client := range service.clientM.clients {
				statis := client.msc.Pumper.Statis()
				totalIn += statis.BytesReaded
				totalOut += statis.BytesWritten
			}
		}()

		cpuPercent, cpuPercentErr := cpu.Percent(0, true)
		if cpuPercentErr != nil {
			timerLog.Error("%s\r\n%s", cpuPercentErr, debug.Stack())
		}
		vmStat, vmError := mem.VirtualMemory()
		if vmError != nil {
			timerLog.Error("%s\r\n%s", vmError, debug.Stack())
		}

		tempInfo := NamingInfo{
			RegisterTime 		: time.Now().UnixNano() / 1000000,
			LoginUsers 			: info.LoginUsers,
			ConnectedClients 	: info.ConnectedClients,
			InData	 			: totalIn - info.InData,
			OutData	 			: totalOut - info.OutData,
			InQueue	 			: info.InQueue,
			OutQueue 			: subscriberClient.stat(service),
			CPU					: cpuPercent,
			MemoryUsage			: vmStat.UsedPercent,
		}

		atomic.StoreInt64(&info.OutData, totalOut)
		atomic.StoreInt64(&info.InData, totalIn)

		jsonText, err := tempInfo.Marshal()
		if err == nil {
			client.HSet("go-servers", getInternetAddress(), jsonText)

			timerLog.Info(fmt.Sprintf("%s", jsonText))
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