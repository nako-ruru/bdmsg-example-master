package connectsvc

import (
	"github.com/farmerx/gorsa"
	"server/connector/internal/config"
	"io/ioutil"
	"strings"
	"encoding/json"
)

type Token struct {
	UserId  	string 		`json:"userId"`		//⽤用户的id
	DeviceId   	string   	`json:"deviceId"`	//⽤用户的设备ID
	Role 		string   	`json:"role"`		//⽤用户⻆角⾊色
	CreateTime	int64 		`json:"createTime"`	//创建时间
	ExpireTime	int64 		`json:"expireTime"`	//过期时间
}

// 初始化设置公钥
func init() {
	if config.Config.PemFile != "" {
		pubPEMData := read1(config.Config.PemFile)
		if err := gorsa.RSA.SetPublicKey(string(pubPEMData)); err != nil {
			panic(err)
		}
	}
}

func decrypt(tokenText string) (Token, error) {
	var token Token
	var err error
	publicKey, _ := gorsa.RSA.GetPublickey()
	splitLength := publicKey.N.BitLen() / 8
	contentBytes := hexToBytes(tokenText)
	arrays := splitBytes(contentBytes, splitLength)
	var jsonText []byte
	for _, array := range arrays {
		var decryptBytes []byte
		decryptBytes, err = gorsa.RSA.PubKeyDECRYPT(array)
		if err != nil {
			return token, err
		}
		jsonText = append(jsonText, decryptBytes...)
	}
	err = json.Unmarshal(jsonText, &token)
	return token, err
}

func read1(path string)[]byte{
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return dat
}

func hexToBytes(hex string)[]byte{
	hexBytes := []byte(hex)
	var result []byte
	for i := 0; i < len(hex); i += 2 {
		ch := hexBytes[i:i+2]
		result = append(result, toByte(ch))
	}
	return result
}

func toByte(ch []byte) byte {
	return byte(strings.Index("0123456789ABCDEF", string(ch[0])) * 16 + strings.Index("0123456789ABCDEF", string(ch[1])))
}

func splitBytes(bytes []byte, splitLength int)[][]byte {
	y := len(bytes) % splitLength	// 余数
	var x int						// 商，数据拆分的组数，余数不为0时+1
	if y != 0 {
		x = len(bytes) / splitLength + 1
	} else {
		x = len(bytes) / splitLength
	}
	arrays := make([][]byte, x)
	for i := 0; i < x; i++ {
		if i == x-1 && len(bytes) % splitLength != 0 {
			arrays[i] = make([]byte, len(bytes) % splitLength)
			arrays[i] = bytes[i*splitLength:]
		} else {
			arrays[i] = make([]byte, splitLength)
			arrays[i] = bytes[i*splitLength : i*splitLength + splitLength]
		}
	}
	return arrays
}