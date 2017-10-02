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

var cachedInternetAddress string

func RegisterNamingService(clientManager *ClientManager)  {
	log.Debug("register %s to %s periodically", getInternetAddress(), config.Config.Redis.Addr)

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

func UnregisterNamingService()  {
	var client = newNamingRedisClient()
	defer client.Close()
	client.HDel("go-servers", getInternetAddress())
}

func newNamingRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.Config.Redis.Addr,
		Password: config.Config.Redis.Password, // no password set
		DB:       config.Config.Redis.Db,           // use default DB
	})
}

func resolveInternetAddress() string {
	return fmt.Sprintf("%s:%d", resolveIp(), getPort())
}

func getInternetAddress() string {
	if cachedInternetAddress == "" {
		cachedInternetAddress = resolveInternetAddress()
	}
	return cachedInternetAddress
}

func resolveIp() string {
	dns := config.Config.IpResolver
	ipVotes := map[string]int{}
	for _, dns := range dns{
		if ip, err := extractIp(dns); err == nil {
			count, ok := ipVotes[ip]
			if !ok {
				count, ipVotes[ip] = 0, 0
			}
			ipVotes[ip] = count + 1
		}
	}

	topVote, topIp := 0, ""
	for ip, vote := range ipVotes {
		if topVote < vote {
			topVote, topIp = vote, ip
		}
	}
	if topIp == "" {
		panic("no available dns")
	} else {
		log.Info("vote internet ip: %v, %s wins", ipVotes, topIp)
		return topIp
	}
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