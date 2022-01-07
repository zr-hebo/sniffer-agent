// +build linux

package capture

import (
	"fmt"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"golang.org/x/net/bpf"
)

func initEthernetHandlerFromPacp() (handler PcapHandler) {
	pcapgoHandler, err := pcapgo.NewEthernetHandle(DeviceName)
	if err != nil {
		panic(fmt.Sprintf("cannot open network interface %s <-- %s", DeviceName, err.Error()))
	}

	// set BPFFilter
	pcapBPF, err := pcap.CompileBPFFilter(
		layers.LinkTypeEthernet, 65535, fmt.Sprintf("tcp and (port %d)", snifferPort))
	if err != nil {
		panic(err.Error())
	}
	bpfIns := []bpf.RawInstruction{}
	for _, ins := range pcapBPF {
		bpfIn := bpf.RawInstruction{
			Op: ins.Code,
			Jt: ins.Jt,
			Jf: ins.Jf,
			K:  ins.K,
		}
		bpfIns = append(bpfIns, bpfIn)
	}

	err = pcapgoHandler.SetBPF(bpfIns)
	if err != nil {
		panic(err.Error())
	}

	_ = pcapgoHandler.SetCaptureLength(65536)
	handler = pcapgoHandler
	return
}
