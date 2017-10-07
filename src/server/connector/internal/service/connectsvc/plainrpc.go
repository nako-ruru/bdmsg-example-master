package connectsvc

import (
	"bytes"
	"compress/zlib"
	"net"
	"sync"
	"server/connector/internal/config"
	"github.com/golang/protobuf/proto"
	"time"
	"github.com/go-redis/redis"
	"sync/atomic"
	"encoding/json"
 	"sort"
	"runtime/debug"
)

var connLocker sync.RWMutex
var availableAddresses []string = []string{}
var callCounter uint64 = 0
var connectionMap map[string]net.Conn = map[string]net.Conn{}

type computeServerInfo struct {
	RegisterTime int64 			`json:"registerTime"`
}

func initRpcServerDiscovery()  {
	ticker := time.NewTicker(time.Second * 1)
	go func() {
		for range ticker.C {
			func() {
				connLocker.Lock()
				defer connLocker.Unlock()

				var client = newComputeServiceRedisClient()
				defer client.Close()

				var command *redis.StringStringMapCmd = client.HGetAll("compute-servers")
				result, err := command.Result()
				if err != nil {
					log.Error("query compute-servers: %s\r\n%s", err, debug.Stack())
				} else {
					availableAddresses = []string{}

					from := time.Now().UnixNano() / 1000000 - 1 * 2000;

					for address, serverInfoText := range result {
						serverInfo := computeServerInfo{}
						bytes := []byte(serverInfoText)
						err := json.Unmarshal(bytes, &serverInfo)
						if err != nil {
							log.Error("json.Unmarshal(bytes, &serverInfo): %s\r\n%s", err, debug.Stack())
						} else if serverInfo.RegisterTime >= from {
							availableAddresses = append(availableAddresses, address)
						}
					}
					if len(availableAddresses) == 0 {
						log.Error("not found compute brokers")
					} else {
						sort.Strings(availableAddresses)
						log.Info("found compute brokers: %s", availableAddresses)
					}
					for address, c := range connectionMap {
						if _, ok := result[address]; !ok {
							c.Close()
							delete(connectionMap, address)
						}
					}
				}
			}()
		}
	}()
}

func deliver(list []*FromConnectorMessage, restCount int, packedMessageId uint64, start int64) {
	if len(list) > 0 {
		msgs := FromConnectorMessages {
			Messages:list,
		}
		bytes, _ := proto.Marshal(&msgs)

		start = time.Now().UnixNano() / 1000000

		compressedBytes := DoZlibCompress(bytes)
		succeed := trySend(compressedBytes, 3, packedMessageId)

		end := time.Now().UnixNano() / 1000000

		if succeed {
			log.Debug("finish consume, packedId=%d, time=%d, cost=%d, msgCount=%d, restCount=%d, uncompressedSize=%d, compressedSize=%d",
				packedMessageId, end, end-start, len(list), restCount, len(bytes), len(compressedBytes))
		} else {
			log.Error("fail consume, packedId=%d, time=%d, cost=%d, msgCount=%d, restCount=%d, uncompressedSize=%d, compressedSize=%d\r\n%s",
				packedMessageId, end, end-start, len(list), restCount, len(bytes), len(compressedBytes), debug.Stack())
		}
	} else {
		log.Debug("finish consume(no messages), packedId=%d", packedMessageId)
	}
}

func trySend(compressedBytes []byte, n int, packedMessageId uint64) bool {
	for k := 0; k < n; k++ {
		err := send(compressedBytes)
		if err == nil {
			return true
		} else {
			log.Error("deliver: n=%d, packedId=%d, %s\r\n%s", k, packedMessageId, err, debug.Stack())
		}
	}
	return false
}

func send(bytes []byte) error {
	connLocker.Lock()
	defer connLocker.Unlock()

	atomic.AddUint64(&callCounter, 1)
	address := availableAddresses[callCounter % uint64(len(availableAddresses))]

	log.Info("callCounter:%d, address: %s", callCounter, address)

	var err error
	var conn, ok = connectionMap[address]

	if !ok {
		if conn != nil {
			conn.Close()
		}
		delete(connectionMap, address)
	}
	if conn == nil {
		conn, err = net.Dial("tcp", address)
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			delete(connectionMap, address)
			return err
		} else {
			connectionMap[address] = conn
		}
	}

	length := len(bytes)
	lengthBytes := []byte{
		byte(length >> 24 & 0xFF),
		byte(length >> 16 & 0xFF),
		byte(length >> 8 & 0xFF),
		byte(length & 0xFF),
	}
	_, err = conn.Write(lengthBytes)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		delete(connectionMap, address)
		return err
	}
	_, err = conn.Write(bytes)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		delete(connectionMap, address)
		return err
	}
	return err
}

var in bytes.Buffer
//进行zlib压缩
func DoZlibCompress(src []byte) []byte {
	in.Reset()
	var w = zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

func newComputeServiceRedisClient() redis.UniversalClient {
	return redis.NewUniversalClient(&redis.UniversalOptions{
		MasterName: 		config.Config.Redis.MasterName,
		Addrs:     			config.Config.Redis.Addresses,
		Password:			config.Config.Redis.Password,
	})
}