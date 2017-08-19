package connectsvc


type RedisMsg struct {
	MessageId string		`json: "messageId"`
	RoomId  string 			`json:"roomId"`
	UserId  string 			`json:"userId"`
	Time    int64			`json:"time"`
	MsgType int 			`json:"type"`
	Nickname string			`json:"nickname"`
	Level int 			`json:"level"`
	Params  map[string]string 	`json:"params"`
}
