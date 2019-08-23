package capture

import (
	"flag"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/bpf"
	"time"

	"github.com/google/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
	"github.com/zr-hebo/sniffer-agent/model"
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
)

var (
	DeviceName  string
	snifferPort int
	inParallel bool
)

func init() {
	flag.StringVar(&DeviceName, "interface", "eth0", "network device name. Default is eth0")
	flag.IntVar(&snifferPort, "port", 3306, "sniffer port. Default is 3306")
	flag.BoolVar(&inParallel, "in_parallel", false, "if capture and deal package in parallel. Default is false")
}

// networkCard is network device
type networkCard struct {
	name string
	listenPort int
	receiver chan model.QueryPiece
}

func NewNetworkCard() (nc *networkCard) {
	// init device
	return &networkCard{
		name: DeviceName,
		listenPort: snifferPort,
		receiver: make(chan model.QueryPiece, 100),
	}
}

func initEthernetHandlerFromPacpgo() (handler *pcapgo.EthernetHandle) {
	handler, err := pcapgo.NewEthernetHandle(DeviceName)
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

	err = handler.SetBPF(bpfIns)
	if err != nil {
		panic(err.Error())
	}

	_ = handler.SetCaptureLength(1024*1024*10)

	return
}

func initEthernetHandlerFromPacp() (handler *pcap.Handle) {
	handler, err := pcap.OpenLive(DeviceName, 65535, false, pcap.BlockForever)
	if err != nil {
		panic(fmt.Sprintf("cannot open network interface %s <-- %s", DeviceName, err.Error()))
	}

	err = handler.SetBPFFilter(fmt.Sprintf("tcp and (port %d)", snifferPort))
	if err != nil {
		panic(err.Error())
	}

	return
}

func (nc *networkCard) Listen() (receiver chan model.QueryPiece) {
	if inParallel {
		nc.listenInParallel()

	} else {
		nc.listenNormal()
	}

	return nc.receiver
}

// Listen get a connection.
func (nc *networkCard) listenNormal() {
	go func() {
		handler := initEthernetHandlerFromPacpgo()
		for {
			var data []byte
			data, ci, err := handler.ZeroCopyReadPacketData()
			if err != nil {
				log.Error(err.Error())
				time.Sleep(time.Second*3)
				continue
			}

			packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.NoCopy)
			m := packet.Metadata()
			m.CaptureInfo = ci
			m.Truncated = m.Truncated || ci.CaptureLength < ci.Length
			nc.parseTCPPackage(packet)
		}
	}()

	return
}

// Listen get a connection.
func (nc *networkCard) listenInParallel() {
	type captureInfo struct {
		bytes []byte
		captureInfo gopacket.CaptureInfo
	}

	rawDataChan := make(chan *captureInfo, 20)
	packageChan := make(chan gopacket.Packet, 20)

	// read packet
	go func() {
		defer func() {
			close(packageChan)
		}()

		handler := initEthernetHandlerFromPacpgo()
		for {
			var data []byte
			// data, ci, err := handler.ZeroCopyReadPacketData()
			data, ci, err := handler.ReadPacketData()
			if err != nil {
				log.Error(err.Error())
				time.Sleep(time.Second*3)
				continue
			}

			rawDataChan <- &captureInfo{
				bytes: data,
				captureInfo: ci,
			}
		}
	}()

	// parse package
	go func() {
		for captureInfo := range rawDataChan {
			packet := gopacket.NewPacket(captureInfo.bytes, layers.LayerTypeEthernet, gopacket.NoCopy)
			m := packet.Metadata()
			m.CaptureInfo = captureInfo.captureInfo
			m.Truncated = m.Truncated || captureInfo.captureInfo.CaptureLength < captureInfo.captureInfo.Length

			packageChan <- packet
		}
	}()

	// parse package
	go func() {
		for packet := range packageChan {
			nc.parseTCPPackage(packet)
		}
	}()

	return
}

func (nc *networkCard) parseTCPPackage(packet gopacket.Packet) {
	var err error
	defer func() {
		if err != nil {
			log.Error("parse TCP package failed <-- %s", err.Error())
		}
	}()

	tcpPkt := packet.TransportLayer().(*layers.TCP)
	if tcpPkt.SYN || tcpPkt.RST {
		return
	}

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		err = fmt.Errorf("no ip layer found in package")
		return
	}

	ipInfo, ok := ipLayer.(*layers.IPv4)
	if !ok {
		err = fmt.Errorf("parsed no ip address")
		return
	}

	// get IP from ip layer
	srcIP := ipInfo.SrcIP.String()
	dstIP := ipInfo.DstIP.String()
	srcPort := int(tcpPkt.SrcPort)
	dstPort := int(tcpPkt.DstPort)
	if dstPort == nc.listenPort {
		// deal mysql server response
		err = readToServerPackage(&srcIP, srcPort, tcpPkt, nc.receiver)
		if err != nil {
			return
		}

	} else if srcPort == nc.listenPort {
		// deal mysql client request
		err = readFromServerPackage(&dstIP, dstPort, tcpPkt)
		if err != nil {
			return
		}
	}

	return
}

func readFromServerPackage(
	srcIP *string, srcPort int, tcpPkt *layers.TCP) (err error) {
	defer func() {
		if err != nil {
			log.Error("read Mysql package send from mysql server to client failed <-- %s", err.Error())
		}
	}()

	if tcpPkt.FIN {
		sessionKey := spliceSessionKey(srcIP, srcPort)
		delete(sessionPool, *sessionKey)
		log.Debugf("close connection from %s", *sessionKey)
		return
	}

	tcpPayload := tcpPkt.Payload
	if (len(tcpPayload) < 1) {
		return
	}

	sessionKey := spliceSessionKey(srcIP, srcPort)
	session := sessionPool[*sessionKey]
	if session != nil {
		// session.readFromServer(tcpPayload)
		// qp = session.GenerateQueryPiece()
		pkt := model.NewTCPPacket(tcpPayload, int64(tcpPkt.Ack), false)
		session.ReceiveTCPPacket(pkt)
	}

	return
}

func readToServerPackage(
	srcIP *string, srcPort int, tcpPkt *layers.TCP, receiver chan model.QueryPiece) (err error) {
	defer func() {
		if err != nil {
			log.Error("read package send from client to mysql server failed <-- %s", err.Error())
		}
	}()

	// when client try close connection remove session from session pool
	if tcpPkt.FIN {
		sessionKey := spliceSessionKey(srcIP, srcPort)
		delete(sessionPool, *sessionKey)
		log.Debugf("close connection from %s", *sessionKey)
		return
	}

	tcpPayload := tcpPkt.Payload
	if (len(tcpPayload) < 1) {
		return
	}

	sessionKey := spliceSessionKey(srcIP, srcPort)
	session := sessionPool[*sessionKey]
	if session == nil {
		session = sd.NewSession(sessionKey, srcIP, srcPort, localIPAddr, snifferPort, receiver)
		sessionPool[*sessionKey] = session
	}

	pkt := model.NewTCPPacket(tcpPayload, int64(tcpPkt.Seq), true)
	session.ReceiveTCPPacket(pkt)

	return
}

