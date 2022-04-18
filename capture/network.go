package capture

import (
	"bufio"
	"bytes"
	"flag"
	"math/rand"
	"time"

	log "github.com/golang/glog"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	pp "github.com/pires/go-proxyproto"
	"github.com/zr-hebo/sniffer-agent/communicator"
	"github.com/zr-hebo/sniffer-agent/model"
	sd "github.com/zr-hebo/sniffer-agent/session-dealer"
)

var (
	DeviceName  string
	snifferPort int
	// inParallel bool
)

func init() {
	flag.StringVar(&DeviceName, "interface", "eth0", "network device name. Default is eth0")
	flag.IntVar(&snifferPort, "port", 3306, "sniffer port. Default is 3306")
	// flag.BoolVar(&inParallel, "in_parallel", false, "if capture and deal package in parallel. Default is false")
}

// networkCard is network device
type networkCard struct {
	name       string
	listenPort int
	receiver   chan model.QueryPiece
}

type PcapHandler interface {
	ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error)
}

func NewNetworkCard() (nc *networkCard) {
	// init device
	return &networkCard{
		name:       DeviceName,
		listenPort: snifferPort,
		receiver:   make(chan model.QueryPiece, 100),
	}
}

func (nc *networkCard) Listen() (receiver chan model.QueryPiece) {
	nc.listenNormal()
	return nc.receiver
}

// Listen get a connection.
func (nc *networkCard) listenNormal() {
	go func() {
		dealTCPPacket := func(srcIP, dstIP string, tcpPkt *layers.TCP, capturePacketRate float64) {
			// send FIN tcp packet to avoid not complete session cannot be released
			// deal FIN packet
			if tcpPkt.FIN {
				nc.parseTCPPackage(srcIP, dstIP, tcpPkt, nil)
				return
			}

			// deal auth packet
			if sd.IsAuthPacket(tcpPkt.Payload) {
				authHeader, _ := pp.Read(bufio.NewReader(bytes.NewReader(tcpPkt.Payload)))
				nc.parseTCPPackage(srcIP, dstIP, tcpPkt, authHeader)
				return
			}

			if 0 < capturePacketRate && capturePacketRate < 1.0 {
				// fall into throw range
				rn := rand.Float64()
				if rn > capturePacketRate {
					return
				}
			}

			nc.parseTCPPackage(srcIP, dstIP, tcpPkt, nil)
		}

		aliveCounter := 0
		dealTCPIPPacket := func(tcpIPPkt *TCPIPPair) {
			// capture packets according to a certain probability
			capturePacketRate := communicator.GetTCPCapturePacketRate()
			if capturePacketRate <= 0 {
				time.Sleep(time.Second * 1)
				aliveCounter += 1
				if aliveCounter >= checkCount {
					aliveCounter = 0
					nc.receiver <- model.NewBaseQueryPiece(localIPAddr, nc.listenPort, capturePacketRate)
				}

			} else {
				dealTCPPacket(tcpIPPkt.srcIP, tcpIPPkt.dstIP, tcpIPPkt.tcpPkt, capturePacketRate)
			}
		}

		dealEachTCPIPPacket(dealTCPIPPacket)
	}()

	return
}

func (nc *networkCard) parseTCPPackage(srcIP, dstIP string, tcpPkt *layers.TCP, authHeader *pp.Header) {
	var err error
	defer func() {
		if err != nil {
			log.Error("parse TCP package failed <-- %s", err.Error())
		}
	}()

	if tcpPkt.SYN || tcpPkt.RST {
		return
	}

	srcPort := int(tcpPkt.SrcPort)
	dstPort := int(tcpPkt.DstPort)

	if dstPort == nc.listenPort {
		// get client ip from proxy auth info
		var clientIP *string
		var clientPort int
		if authHeader != nil && authHeader.SourceAddress.String() != srcIP {
			clientIPContent := authHeader.SourceAddress.String()
			clientIP = &clientIPContent
			clientPort = int(authHeader.SourcePort)

		} else {
			clientIP = &srcIP
			clientPort = srcPort
		}

		// deal mysql server response
		err = readToServerPackage(clientIP, clientPort, &dstIP, tcpPkt, nc.receiver)
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
	clientIP *string, clientPort int, tcpPkt *layers.TCP) (err error) {
	defer func() {
		if err != nil {
			log.Error("read Mysql package send from mysql server to client failed <-- %s", err.Error())
		}
	}()

	if tcpPkt.FIN {
		sessionKey := spliceSessionKey(clientIP, clientPort)
		session := sessionPool[*sessionKey]
		if session != nil {
			session.Close()
			delete(sessionPool, *sessionKey)
		}
		return
	}

	tcpPayload := tcpPkt.Payload
	if len(tcpPayload) < 1 {
		return
	}

	sessionKey := spliceSessionKey(clientIP, clientPort)
	session := sessionPool[*sessionKey]
	if session != nil {
		pkt := model.NewTCPPacket(tcpPayload, int64(tcpPkt.Ack), false)
		session.ReceiveTCPPacket(pkt)
	}

	return
}

func readToServerPackage(
	clientIP *string, clientPort int, destIP *string, tcpPkt *layers.TCP,
	receiver chan model.QueryPiece) (err error) {
	defer func() {
		if err != nil {
			log.Error("read package send from client to mysql server failed <-- %s", err.Error())
		}
	}()

	// when client try close connection remove session from session pool
	if tcpPkt.FIN {
		sessionKey := spliceSessionKey(clientIP, clientPort)
		session := sessionPool[*sessionKey]
		if session != nil {
			session.Close()
			delete(sessionPool, *sessionKey)
		}
		log.Infof("close connection from %s", *sessionKey)
		return
	}

	tcpPayload := tcpPkt.Payload
	if len(tcpPayload) < 1 {
		return
	}

	sessionKey := spliceSessionKey(clientIP, clientPort)
	session := sessionPool[*sessionKey]
	if session == nil {
		session = sd.NewSession(sessionKey, clientIP, clientPort, destIP, snifferPort, receiver)
		sessionPool[*sessionKey] = session
	}

	pkt := model.NewTCPPacket(tcpPayload, int64(tcpPkt.Seq), true)
	session.ReceiveTCPPacket(pkt)

	return
}
