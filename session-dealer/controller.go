package session_dealer

import (
	"github.com/zr-hebo/sniffer-agent/model"
	"github.com/zr-hebo/sniffer-agent/session-dealer/mysql"
)

func NewSession(sessionKey *string, clientIP *string, clientPort int, serverIP *string, serverPort int,
	receiver chan model.QueryPiece) (session ConnSession) {
	switch serviceType {
	case ServiceTypeMysql:
		session = mysql.NewMysqlSession(sessionKey, clientIP, clientPort, serverIP, serverPort, receiver)
	default:
		session = mysql.NewMysqlSession(sessionKey, clientIP, clientPort, serverIP, serverPort, receiver)
	}
	return
}

func CheckParams()  {
	switch serviceType {
	case ServiceTypeMysql:
		mysql.CheckParams()
	default:
		mysql.CheckParams()
	}
}

func IsAuthPacket(payload []byte) bool {
	switch serviceType {
	case ServiceTypeMysql:
		return len(payload) >= 5 && mysql.IsAuth(payload[4])

	default:
		return false
	}
}
