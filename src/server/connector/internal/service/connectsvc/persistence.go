package connectsvc


type RedisMsg struct {
	RoomId  string
	UserId  string
	Time    int64
	MsgType int
	Params  map[string]string
}
