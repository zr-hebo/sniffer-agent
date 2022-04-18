//go:build windows
// +build windows

package capture

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	handler *pcap.Handle
)

func initEthernetHandlerFromPacp() (pcapHandler *pcap.Handle) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range devices {
		log.Infof("found Windows device:'%s', device info:%s", device.Name, device.Description)
	}

	pcapHandler, err = pcap.OpenLive(DeviceName, 1024, false, time.Hour*24)
	if err != nil {
		panic(fmt.Sprintf("cannot open network interface %s <-- %s", DeviceName, err.Error()))
	}

	return
}

func dealEachTCPIPPacket(dealTCPIPPacket func(tcpIPPkt *TCPIPPair)) {
	handler = initEthernetHandlerFromPacp()
	defer handler.Close()
	packetSource := gopacket.NewPacketSource(handler, handler.LinkType())
	for packet := range packetSource.Packets() {
		if err := packet.ErrorLayer(); err != nil {
			log.Error(err.Error())
			continue
		}

		// Process packet here
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			continue
		}
		tcpPkt := tcpLayer.(*layers.TCP)
		if (int(tcpPkt.SrcPort) != snifferPort && int(tcpPkt.DstPort) != snifferPort) {
			continue
		}

		var srcIP, dstIP string
		ipLayer := packet.NetworkLayer()
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
	return
}
