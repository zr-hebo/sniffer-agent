package session_dealer

import (
	"github.com/zr-hebo/sniffer-agent/session-dealer/mysql"
)

func NewSession(sessionKey *string, clientIP *string, clientPort int, serverIP *string, serverPort int) (session ConnSession) {
	switch serviceType {
	case ServiceTypeMysql:
		session = mysql.NewMysqlSession(sessionKey, clientIP, clientPort, serverIP, serverPort)
	default:
		session = mysql.NewMysqlSession(sessionKey, clientIP, clientPort, serverIP, serverPort)
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
