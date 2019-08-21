package model

type TCPPacket struct {
	Payload []byte
	Seq int64
	ToServer bool
}

func NewTCPPacket(payload []byte, seq int64, toServer bool) *TCPPacket {
	return &TCPPacket{
		Payload: payload,
		Seq: seq,
		ToServer: toServer,
	}
}
