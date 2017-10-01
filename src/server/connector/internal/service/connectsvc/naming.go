package connectsvc

import (
	"github.com/go-redis/redis"
	"time"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"server/connector/internal/config"
	"fmt"
)


type NamingInfo struct {
	RegisterTime     int64 `json:"registerTime"`
	LoginUsers       int   `json:"loginUsers"`
	ConnectedClients int   `json:"connectedClients"`
}

func RegisterNamingService(clientManager *ClientManager)  {
	host := getHost()
	log.Info("register %s to %s periodically", host, config.Config.Redis.Addr)

	var f = func() {
		var client = newNamingRedisClient()
		defer client.Close()


		clientManager.locker.Lock()
		defer clientManager.locker.Unlock()
		info := NamingInfo{
			RegisterTime:     time.Now().UnixNano() / 1000000,
			LoginUsers:       len(clientManager.clients),
			ConnectedClients: len(clientManager.clients),
		}
		jsonText, err := info.Marshal()
		if err == nil {
			client.HSet("go-servers", host, jsonText)
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

func UnregisterNamingService()  {
	var client = newNamingRedisClient()
	defer client.Close()
	client.HDel("go-servers", getHost())
}

func newNamingRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.Config.Redis.Addr,
		Password: config.Config.Redis.Password, // no password set
		DB:       config.Config.Redis.Db,           // use default DB
	})
}

func getHost() string {
	return fmt.Sprintf("%s:%d", getIp(), getPort())
}

func getIp() string {
	dns := [] string{"http://2017.ip138.com/ic.asp", "http://members.3322.org/dyndns/getip"}
	for i := 0; i < len(dns); i++ {
		if ip, err := extractIp(dns[i]); err == nil {
			return ip
		}
	}
	panic("no available dns")
}

func extractIp(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	re := regexp.MustCompile(`(\d{1,3}\.){3}\d{1,3}`)
	sprint := string(body)
	ip := re.FindString(sprint)
	return ip, nil
}

func getPort() int {
	return config.Config.ServiceS.Connect.ListenAddr.Port
}

func (p *NamingInfo) Marshal() ([]byte, error) {
	return json.Marshal(p)
}