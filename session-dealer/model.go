package session_dealer

import "github.com/zr-hebo/sniffer-agent/model"

type ConnSession interface {
	ReceiveTCPPacket(*model.TCPPacket)
}
