package mysql

import (
	"fmt"
)

const (
	PACKET_OK = 0
	PACKET_EOF = 254
	PACKET_ERR = 255
)

func lenencInt(bytesVal []byte) (val int64) {
	if bytesVal == nil || len(bytesVal) < 1 {
		val = -1
		return
	}

	fb := bytesVal[0]
	var offset int64
	switch {
	case fb < 251:
		val = int64(fb)

	case fb == 252:
		numLen := int64(2)
		offset = 1+numLen
		val = int64(bytesToInt(bytesVal[1:offset]))

	case fb == 253:
		numLen := int64(3)
		offset = 1+numLen
		val = int64(bytesToInt(bytesVal[1:offset]))

	case fb == 254:
		numLen := int64(8)
		offset = 1+numLen
		val = int64(bytesToInt(bytesVal[1:offset]))

	default:
		val = -1
	}
	return
}

func parseResponseHeader(payload []byte) (ok, val int64, err error) {
	if payload == nil || len(payload) < 1 {
		err = fmt.Errorf("no bytes to parse")
		return
	}

	fmt.Printf("%#v\n", payload)
	defer func() {
		fmt.Printf("%#v\n", ok)
		fmt.Printf("%#v\n", val)
	}()

	switch {
	case payload[0] == PACKET_OK && len(payload)>=7:
	case payload[0] == PACKET_EOF && len(payload)<=9:
		// set ok and mysql affected rows number
		ok = 1
		val = lenencInt(payload)

	case payload[0] == PACKET_ERR && len(payload)>3:
		// set not ok and mysql execute error-code
		ok = 0
		val = int64(bytesToIntSmallEndian(payload[1:3]))

	default:
		err = fmt.Errorf("invalid response packet")
	}
	return
}

