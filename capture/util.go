package capture

import (
	"fmt"
	"net"
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
	// var buf strings.Builder
	// _, err := fmt.Fprint(&buf, *srcIP, ":", srcPort)
	// if err != nil {
	// 	panic(err.Error())
	// }
	// sessionKey := buf.String()

	// buf := new(bytes.Buffer)
	// _ = templateSessionKey.ExecuteTemplate(buf, "IP", srcIP)
	// _ = templateSessionKey.ExecuteTemplate(buf, "Port", srcPort)
	// sessionKey := buf.String()

	sessionKey := fmt.Sprintf("%s:%d", *srcIP, srcPort)
	return &sessionKey
}
