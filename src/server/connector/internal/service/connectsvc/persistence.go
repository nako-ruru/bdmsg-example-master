package connectsvc


type ToComputeMessage struct {
	MessageId 	string				`json:"messageId"`
	RoomId  	string 				`json:"roomId"`
	UserId  	string 				`json:"userId"`
	Nickname 	string				`json:"nickname"`
	Level 		int 				`json:"level"`
	MsgType 	int 				`json:"type"`
	Params  	map[string]string 	`json:"params"`
	Time    	int64				`json:"time"`
}
