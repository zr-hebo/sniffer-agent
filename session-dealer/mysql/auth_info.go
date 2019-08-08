package mysql

// parseAuthInfo parse username, dbname from mysql client auth info
func parseAuthInfo(data []byte) (userName, dbName string, err error) {
	var resp handshakeResponse41
	pos, err := parseHandshakeResponseHeader(&resp, data)
	if err != nil {
		return
	}

	// Read the remaining part of the packet.
	if err = parseHandshakeResponseBody(&resp, data, pos); err != nil {
		return
	}

	userName = resp.User
	dbName = resp.DBName
	return
}
