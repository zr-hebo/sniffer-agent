//go:build darwin
// +build darwin

package capture

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// in online use, we found a strange bug: pcap cost 100% core CPU and memory increase along
func initEthernetHandlerFromPacp() (pcapHandler *pcap.Handle) {
	pcapHandler, err := pcap.OpenLive(DeviceName, 65536, false, pcap.BlockForever)
	if err != nil {
		panic(fmt.Sprintf("cannot open network interface %s <-- %s", DeviceName, err.Error()))
	}

	err = pcapHandler.SetBPFFilter(fmt.Sprintf("tcp and (port %d)", snifferPort))
	if err != nil {
		panic(err.Error())
	}

	return
}

func dealEachTCPIPPacket(dealTCPIPPacket func(tcpIPPkt *TCPIPPair)) {
	handler := initEthernetHandlerFromPacp()
	defer func() {
		handler.Close()
	}()

	for {
		var ci gopacket.CaptureInfo
		data, ci, err := handler.ZeroCopyReadPacketData()
		if err != nil {
			log.Error(err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.NoCopy)
		m := packet.Metadata()
		m.CaptureInfo = ci

		tcpPkt, ok := packet.TransportLayer().(*layers.TCP)
		if !ok {
			continue
		}

		ipLayer := packet.NetworkLayer()
		if ipLayer == nil {
			log.Error("no ip layer found in package")
			continue
		}

		var srcIP, dstIP string
		switch realIPLayer := ipLayer.(type) {
		case *layers.IPv6:
			{
				srcIP = realIPLayer.SrcIP.String()
				dstIP = realIPLayer.DstIP.String()
			}
		case *layers.IPv4:
			{
				srcIP = realIPLayer.SrcIP.String()
				dstIP = realIPLayer.DstIP.String()
			}
		}

		tcpipPair := &TCPIPPair{
			srcIP:  srcIP,
			dstIP:  dstIP,
			tcpPkt: tcpPkt,
		}
		dealTCPIPPacket(tcpipPair)
	}
}
