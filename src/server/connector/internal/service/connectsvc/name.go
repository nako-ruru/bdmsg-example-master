package connectsvc

import (
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

func Register2()  {
	//注册zk节点q
	conn, err := getConnect()
	if err != nil {
		log.Error(" connect zk error: %s ", err)
	}
	defer conn.Close()
	err = registServer(conn, "47.92.68.14:6600")
	if err != nil {
		log.Error(" regist node error: %s ", err)
	}
}


func getConnect() (conn *zk.Conn, err error) {
	conn, _, err = zk.Connect([]string{"47.92.68.14:2181"}, 5 * time.Second)
	if err != nil {
		log.Error("GetConnect: %s", err)
	}
	return
}

func registServer(conn *zk.Conn, host string) (err error) {
	_, err = conn.Create("/go_servers/"+host, nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	return
}

func getServerList(conn *zk.Conn) (list []string, err error) {
	list, _, err = conn.Children("/go_servers")
	return
}