package capture

import (
	"bytes"
	"fmt"
	"github.com/pingcap/tidb/util/hack"
	"net"
	"strconv"
	"strings"
)

func getLocalIPAddr() (ipAddr string, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, addr := range addrs {
		addrStr := addr.String()
		if strings.Contains(addrStr, "127.0.0.1") ||
			strings.Contains(addrStr, "::1") ||
			strings.Contains(addrStr, "/64") {
			continue
		}

		addrStr = strings.TrimRight(addrStr, "1234567890")
		addrStr = strings.TrimRight(addrStr, "/")
		if len(addrStr) < 1 {
			continue
		}

		ipAddr = addrStr
		return
	}

	err = fmt.Errorf("no valid ip address found")
	return
}

func spliceSessionKey(srcIP *string, srcPort int) (*string) {
	// sessionKey := fmt.Sprintf("%s:%d", *srcIP, srcPort)
	var buffer = bytes.NewBuffer(make([]byte, 0, 24))
	buffer.WriteString(*srcIP)
	buffer.WriteString(":")
	buffer.WriteString(strconv.Itoa(srcPort))
	sessionKey := hack.String(buffer.Bytes())
	return &sessionKey
}
