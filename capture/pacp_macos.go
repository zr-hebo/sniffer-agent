// +build darwin

package capture

import (
	"fmt"

	"github.com/google/gopacket/pcap"
)

// in online use, we found a strange bug: pcap cost 100% core CPU and memory increase along
func initEthernetHandlerFromPacp() (handler PcapHandler) {
	pcapHandler, err := pcap.OpenLive(DeviceName, 65536, false, pcap.BlockForever)
	if err != nil {
		panic(fmt.Sprintf("cannot open network interface %s <-- %s", DeviceName, err.Error()))
	}

	err = pcapHandler.SetBPFFilter(fmt.Sprintf("tcp and (port %d)", snifferPort))
	if err != nil {
		panic(err.Error())
	}

	handler = pcapHandler
	return
}
