package capture

import (
	"github.com/google/gopacket/layers"
)

type TCPIPPair struct {
	srcIP  string
	dstIP  string
	tcpPkt *layers.TCP
}
