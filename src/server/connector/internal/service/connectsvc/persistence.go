package connectsvc


type RedisMsg struct {
	RoomId  string 			`json:"roomId"`
	UserId  string 			`json:"userId"`
	Time    int64			`json:"time"`
	MsgType int 			`json:"type"`
	Params  map[string]string 	`json:"params"`
}
