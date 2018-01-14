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
var rpcClient = rpc{
	connLocker: 		sync.RWMutex{},
	availableAddresses:	[]string{},
	callCounter:		0,
	connectionMap: 		map[string]net.Conn{},
}

func initRpcServerDiscovery()  {
	ticker := time.NewTicker(time.Second * 1)
	f := func() {
		var result map[string]string
		var err error

		func() {
			var client = newComputeServiceRedisClient()
			defer client.Close()

			command := client.HGetAll("compute-servers")
			result, err = command.Result()
		}()

		log.Trace("1000000")
		if err != nil {
			log.Trace("2000000")
			log.Error("query compute-servers: %s\r\n%s", err, debug.Stack())
		} else {
			var availableAddresses []string

			log.Trace("3000000")

			from := time.Now().UnixNano() / 1000000 - 1 * 2000

			log.Trace("4000000")
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

			log.Trace("5000000")
			if len(availableAddresses) == 0 {
				log.Error("no compute brokers found")
			} else {
				sort.Strings(availableAddresses)
				timerLog.Info("compute brokers found: %s", availableAddresses)
			}
			log.Trace("6000000")

			func() {
				rpcClient.connLocker.Lock()
				defer rpcClient.connLocker.Unlock()

				rpcClient.availableAddresses = availableAddresses
				for address, c := range rpcClient.connectionMap {
					if _, ok := result[address]; !ok {
						if c != nil {
							c.Close()
						}
					}
				}
				log.Trace("7000000")
			}()
		}
	}
	go func() {
		for range ticker.C {
			f()
		}
	}()
	f()
}

func (rpc rpc) deliver(list []*FromConnectorMessage, restCount int, packedMessageId uint64, start int64) {
	log.Trace("71000000000")
	if len(list) > 0 {
		log.Trace("72000000000")
		msgs := FromConnectorMessages {
			Messages:list,
		}
		bytes, _ := proto.Marshal(&msgs)

		start = time.Now().UnixNano() / 1000000

		log.Trace("73000000000")
		compressedBytes := rpc.DoZlibCompress(bytes)
		log.Trace("73100000000")
		succeed := rpc.trySend(compressedBytes, 3, packedMessageId)
		log.Trace("73200000000")
		end := time.Now().UnixNano() / 1000000

		log.Trace("74000000000")
		if succeed {
			log.Trace("75000000000")
			log.Debug("finish consume, packedId=%d, time=%d, cost=%d, msgCount=%d, restCount=%d, uncompressedSize=%d, compressedSize=%d",
				packedMessageId, end, end-start, len(list), restCount, len(bytes), len(compressedBytes))
		} else {
			log.Trace("76000000000")
			log.Error("fail consume, packedId=%d, time=%d, cost=%d, msgCount=%d, restCount=%d, uncompressedSize=%d, compressedSize=%d\r\n%s",
				packedMessageId, end, end-start, len(list), restCount, len(bytes), len(compressedBytes), debug.Stack())
		}
		log.Trace("77000000000")
	} else {
		log.Trace("78000000000")
		log.Debug("finish consume(no messages), packedId=%d", packedMessageId)
		log.Trace("79000000000")
	}
}

func (rpc rpc) trySend(compressedBytes []byte, n int, packedMessageId uint64) bool {
	log.Trace("73110000000")
	for k := 0; k < n; k++ {
		log.Trace("73120000000")
		err := rpc.send(compressedBytes)
		log.Trace("73130000000")
		if err == nil {
			log.Trace("73140000000")
			return true
		} else {
			log.Trace("73150000000")
			log.Error("deliver: n=%d, packedId=%d, %s\r\n%s", k, packedMessageId, err, debug.Stack())
		}
		log.Trace("73160000000")
	}
	log.Trace("73170000000")
	return false
}

func (rpc rpc)  send(bytes []byte) error {
	log.Trace("73121000000")
	rpc.connLocker.Lock()
	log.Trace("73122000000")
	defer rpc.connLocker.Unlock()

	log.Trace("73123000000")
	atomic.AddUint64(&rpc.callCounter, 1)
	address := rpc.availableAddresses[rpc.callCounter % uint64(len(rpc.availableAddresses))]

	log.Trace("73124000000")
	log.Debug("callCounter:%d, address: %s", rpc.callCounter, address)

	var err error
	var conn, ok = rpc.connectionMap[address]

	log.Trace("73125000000")
	if !ok {
		if conn != nil {
			conn.Close()
		}
		delete(rpc.connectionMap, address)
	}
	log.Trace("73126000000")
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

	log.Trace("73127000000")
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
	log.Trace("73128000000")
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