package capture

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
	"github.com/zr-hebo/sniffer-agent/model"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	DeviceName  string
	snifferPort int
)

func init() {
	flag.StringVar(&DeviceName, "interface", "eth0", "network device name. Default is eth0")
	flag.IntVar(&snifferPort, "port", 3306, "sniffer port. Default is 3306")
}

// networkCard is network device
type networkCard struct {
	name string
	listenPort int
}

func NewNetworkCard() (nc *networkCard) {
	// init device
	return &networkCard{name: DeviceName, listenPort: snifferPort}
}

// Listen get a connection.
func (nc *networkCard) Listen() (receiver chan model.QueryPiece) {
	receiver = make(chan model.QueryPiece, 100)

	go func() {
		defer func() {
			close(receiver)
		}()

		handle, err := pcap.OpenLive(DeviceName, 65535, false, pcap.BlockForever)
		if err != nil {
			panic(fmt.Sprintf("cannot open network interface %s <-- %s", nc.name, err.Error()))
		}

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		for packet := range packetSource.Packets() {
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil {
				// log.Info("empty network layer")
				continue
			}

			if packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				// log.Info("packet type is %s, not TCP", packet.TransportLayer().LayerType())
				continue
			}

			qp := nc.parseTCPPackage(packet)
			if qp != nil {
				receiver <- qp
			}
		}
	}()

	return
}

func (nc *networkCard) parseTCPPackage(packet gopacket.Packet) (qp model.QueryPiece) {
	var err error
	defer func() {
		if err != nil {
			log.Error("parse TCP package failed <-- %s", err.Error())
		}
	}()

	tcpConn := packet.TransportLayer().(*layers.TCP)
	if tcpConn.SYN || tcpConn.RST {
		return
	}

	if(int(tcpConn.DstPort) != nc.listenPort && int(tcpConn.SrcPort) != nc.listenPort) {
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
	srcPort := int(tcpConn.SrcPort)
	dstPort := int(tcpConn.DstPort)
	if dstIP == localIPAddr && dstPort == nc.listenPort {
		// deal mysql server response
		err = readToServerPackage(srcIP, srcPort, tcpConn)
		if err != nil {
			return
		}

	} else if srcIP == localIPAddr && srcPort == nc.listenPort {
		// deal mysql client request
		qp, err = readFromServerPackage(dstIP, dstPort, tcpConn)
		if err != nil {
			return
		}
	}

	return
}

func readFromServerPackage(srcIP string, srcPort int, tcpConn *layers.TCP) (qp model.QueryPiece, err error) {
	defer func() {
		if err != nil {
			log.Error("read Mysql package send from mysql server to client failed <-- %s", err.Error())
		}
	}()

	sessionKey := spliceSessionKey(srcIP, srcPort)
	if tcpConn.FIN {
		delete(sessionPool, sessionKey)
		// log.Debugf("close connection from %s", sessionKey)
		return
	}

	tcpPayload := tcpConn.Payload
	if (len(tcpPayload) < 1) {
		return
	}

	session := sessionPool[sessionKey]
	if session != nil {
		session.ReadFromServer(tcpPayload)
		qp = session.GenerateQueryPiece()
	}

	return
}

func readToServerPackage(srcIP string, srcPort int, tcpConn *layers.TCP) (err error) {
	defer func() {
		if err != nil {
			log.Error("read package send from client to mysql server failed <-- %s", err.Error())
		}
	}()

	sessionKey := spliceSessionKey(srcIP, srcPort)
	// when client try close connection remove session from session pool
	if tcpConn.FIN {
		delete(sessionPool, sessionKey)
		// log.Debugf("close connection from %s", sessionKey)
		return
	}

	tcpPayload := tcpConn.Payload
	if (len(tcpPayload) < 1) {
		return
	}

	session := sessionPool[sessionKey]
	if session == nil {
		session = sd.NewSession(sessionKey, srcIP, srcPort, localIPAddr, snifferPort)
		sessionPool[sessionKey] = session
	}

	session.ReadFromClient(tcpPayload)
	return
}

