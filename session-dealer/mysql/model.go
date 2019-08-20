package mysql

type handshakeResponse41 struct {
	Capability uint32
	Collation  uint8
	User       string
	DBName     string
	Auth       []byte
}

// jigsaw record tcp package begin and end seq id
type jigsaw struct {
	b int64
	e int64
}
