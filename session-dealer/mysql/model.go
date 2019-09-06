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
	begin int64
	end   int64
}

func newJigsaw(begin, end int64) (js *jigsaw) {
	return &jigsaw{
		begin: begin,
		end: end,
	}
}