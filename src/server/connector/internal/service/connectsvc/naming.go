package connectsvc

import (
	"github.com/go-redis/redis"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"server/connector/internal/config"
	"fmt"
)



var cachedInternetAddress string


func UnregisterNamingService()  {
	var client = newNamingRedisClient()
	defer client.Close()
	client.HDel("go-servers", getInternetAddress())
}

func newNamingRedisClient() redis.UniversalClient {
	return redis.NewUniversalClient(&redis.UniversalOptions{
		MasterName: config.Config.Redis.MasterName,
		Addrs:      config.Config.Redis.Addresses,
		Password:			config.Config.Redis.Password,
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