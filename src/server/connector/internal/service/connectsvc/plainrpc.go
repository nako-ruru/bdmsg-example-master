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

type computeServerInfo struct {
	RegisterTime int64 			`json:"registerTime"`
}
type rpc struct {
	connLocker 			sync.RWMutex
	availableAddresses 	[]string
	callCounter 		uint64
	connectionMap 		map[string]net.Conn

	in bytes.Buffer
}
var rpcClient rpc = rpc{
	connLocker: 		sync.RWMutex{},
	availableAddresses:	[]string{},
	callCounter:		0,
	connectionMap: 		map[string]net.Conn{},
}

func initRpcServerDiscovery()  {
	ticker := time.NewTicker(time.Second * 1)
	go func() {
		for range ticker.C {
			func() {
				rpcClient.connLocker.Lock()
				defer rpcClient.connLocker.Unlock()

				var client = newComputeServiceRedisClient()
				defer client.Close()

				var command *redis.StringStringMapCmd = client.HGetAll("compute-servers")
				result, err := command.Result()
				if err != nil {
					log.Error("query compute-servers: %s\r\n%s", err, debug.Stack())
				} else {
					rpcClient.availableAddresses = []string{}

					from := time.Now().UnixNano() / 1000000 - 1 * 2000;

					for address, serverInfoText := range result {
						serverInfo := computeServerInfo{}
						bytes := []byte(serverInfoText)
						err := json.Unmarshal(bytes, &serverInfo)
						if err != nil {
							log.Error("json.Unmarshal(bytes, &serverInfo): %s\r\n%s", err, debug.Stack())
						} else if serverInfo.RegisterTime >= from {
							rpcClient.availableAddresses = append(rpcClient.availableAddresses, address)
						}
					}
					if len(rpcClient.availableAddresses) == 0 {
						log.Error("no compute brokers found")
					} else {
						sort.Strings(rpcClient.availableAddresses)
						timerLog.Info("compute brokers found: %s", rpcClient.availableAddresses)
					}
					for address, c := range rpcClient.connectionMap {
						if _, ok := result[address]; !ok {
							c.Close()
							delete(rpcClient.connectionMap, address)
						}
					}
				}
			}()
		}
	}()
}

func (rpc rpc) deliver(list []*FromConnectorMessage, restCount int, packedMessageId uint64, start int64) {
	if len(list) > 0 {
		msgs := FromConnectorMessages {
			Messages:list,
		}
		bytes, _ := proto.Marshal(&msgs)

		start = time.Now().UnixNano() / 1000000

		compressedBytes := rpc.DoZlibCompress(bytes)
		succeed := rpc.trySend(compressedBytes, 3, packedMessageId)

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

func (rpc rpc) trySend(compressedBytes []byte, n int, packedMessageId uint64) bool {
	for k := 0; k < n; k++ {
		err := rpc.send(compressedBytes)
		if err == nil {
			return true
		} else {
			log.Error("deliver: n=%d, packedId=%d, %s\r\n%s", k, packedMessageId, err, debug.Stack())
		}
	}
	return false
}

func (rpc rpc)  send(bytes []byte) error {
	rpc.connLocker.Lock()
	defer rpc.connLocker.Unlock()

	atomic.AddUint64(&rpc.callCounter, 1)
	address := rpc.availableAddresses[rpc.callCounter % uint64(len(rpc.availableAddresses))]

	log.Debug("callCounter:%d, address: %s", rpc.callCounter, address)

	var err error
	var conn, ok = rpc.connectionMap[address]

	if !ok {
		if conn != nil {
			conn.Close()
		}
		delete(rpc.connectionMap, address)
	}
	if conn == nil {
		conn, err = net.Dial("tcp", address)
		if err != nil {
			if conn != nil {
				conn.Close()
			}
			delete(rpc.connectionMap, address)
			return err
		} else {
			rpc.connectionMap[address] = conn
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
		delete(rpc.connectionMap, address)
		return err
	}
	_, err = conn.Write(bytes)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		delete(rpc.connectionMap, address)
		return err
	}
	return err
}

//进行zlib压缩
func (rpc rpc) DoZlibCompress(src []byte) []byte {
	rpc.in.Reset()
	var w = zlib.NewWriter(&rpc.in)
	w.Write(src)
	w.Close()
	return rpc.in.Bytes()
}

func newComputeServiceRedisClient() redis.UniversalClient {
	return redis.NewUniversalClient(&redis.UniversalOptions{
		MasterName: 		config.Config.Redis.MasterName,
		Addrs:     			config.Config.Redis.Addresses,
		Password:			config.Config.Redis.Password,
	})
}