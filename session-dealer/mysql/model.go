package mysql

type handshakeResponse41 struct {
	Capability uint32
	Collation  uint8
	User       string
	DBName     string
	Auth       []byte
}

// receiveRange record mysql package begin and end seq id
type receiveRange struct {
	beginSeqID int64
	endSeqID   int64
}
