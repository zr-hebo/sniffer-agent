package mysql

import (
	"fmt"
	"strings"
	"time"

	"github.com/siddontang/go/hack"
	log "github.com/sirupsen/logrus"
	"github.com/zr-hebo/sniffer-agent/model"
)

type MysqlSession struct {
	connectionID      *string
	visitUser         *string
	visitDB           *string
	clientHost        *string
	clientPort        int
	serverIP          *string
	serverPort        int
	stmtBeginTime     int64
	beginSeqID        int64
	packageOffset     int64
	expectReceiveSize int
	coverRanges       []*jigsaw
	expectSendSize    int
	prepareInfo       *prepareInfo
	cachedPrepareStmt map[int]*string
	cachedStmtBytes   []byte
	computeWindowSizeCounter int

	tcpPacketCache []*model.TCPPacket

	queryPieceReceiver chan model.QueryPiece
	lastSeq int64
	keepAlive chan bool

	ackID int64
	sendSize    int64
}

type prepareInfo struct {
	prepareStmtID int
}

var (
	windowSizeCache = make(map[string]*packageWindowCounter, 16)
)

const (
	maxIPPackageSize = 1 << 16
)

func NewMysqlSession(
	sessionKey *string, clientIP *string, clientPort int, serverIP *string, serverPort int,
	receiver chan model.QueryPiece) (ms *MysqlSession) {
	ms = &MysqlSession{
		connectionID:      sessionKey,
		clientHost:        clientIP,
		clientPort:        clientPort,
		serverIP:          serverIP,
		serverPort:        serverPort,
		stmtBeginTime:     time.Now().UnixNano() / millSecondUnit,
		cachedPrepareStmt: make(map[int]*string, 8),
		coverRanges: make([]*jigsaw, 0, 4),
		queryPieceReceiver: receiver,
		keepAlive: make(chan bool, 1),
		lastSeq: -1,
		ackID: -1,
		sendSize: 0,
	}

	go ms.haha()

	return
}

func (ms *MysqlSession) Stop() {
	ms.keepAlive <- false
}

func (ms *MysqlSession) haha() {

	for true {
		// select {
		// case  <- ms.keepAlive:
		// 	return
		// default:
		// }

		if len(ms.tcpPacketCache) < 1 {
			// log.Debugf("there are %d packages in tcp packet cache", )
			time.Sleep(1)
			continue
		}

		beginIdx := -1
		if ms.lastSeq < 0 {
			ms.lastSeq = ms.tcpPacketCache[0].Seq
		}
		for idx := 0; idx < len(ms.tcpPacketCache); idx++  {
			pkt := ms.tcpPacketCache[idx]
			if ms.lastSeq == pkt.Seq {
				beginIdx = idx
				ms.lastSeq = pkt.Seq + int64(len(pkt.Payload))

			} else {
				break
			}
		}


		if beginIdx < 0 {
			return
		}

		inOrderPkgs := ms.tcpPacketCache[:beginIdx+1]
		if beginIdx == len(ms.tcpPacketCache) - 1 {
			ms.tcpPacketCache = make([]*model.TCPPacket, 0, 4)
		} else {
			ms.tcpPacketCache = ms.tcpPacketCache[beginIdx+1:]
		}

		for _, pkg := range inOrderPkgs {
			if pkg.ToServer {
				ms.ReadFromClient(pkg.Seq, pkg.Payload)

			} else {
				ms.ReadFromServer(pkg.Payload)
				ms.queryPieceReceiver <- ms.GenerateQueryPiece()
			}
		}
	}
}

func (ms *MysqlSession) ReceiveTCPPacket(newPkt *model.TCPPacket)  {
	if !newPkt.ToServer && ms.ackID + ms.sendSize == newPkt.Seq {
		// ignore to response to client data
		ms.ackID = ms.ackID + newPkt.Seq
		ms.sendSize = ms.sendSize + int64(len(newPkt.Payload))
		return

	} else if !newPkt.ToServer {
		ms.ackID = newPkt.Seq
		ms.sendSize = int64(len(newPkt.Payload))
	}

	insertIdx := len(ms.tcpPacketCache)
	for idx, pkt := range ms.tcpPacketCache {
		if pkt.Seq > newPkt.Seq {
			insertIdx = idx
		}
	}

	if insertIdx == len(ms.tcpPacketCache) {
		ms.tcpPacketCache = append(ms.tcpPacketCache, newPkt)
	} else {
		newCache := make([]*model.TCPPacket, len(ms.tcpPacketCache)+1)
		copy(newCache[:insertIdx], ms.tcpPacketCache[:insertIdx])
		newCache[insertIdx] = newPkt
		copy(newCache[insertIdx+1:], ms.tcpPacketCache[insertIdx:])
	}
}

func (ms *MysqlSession) ResetBeginTime()  {
	ms.stmtBeginTime = time.Now().UnixNano() / millSecondUnit
}

func (ms *MysqlSession) ReadFromServer(bytes []byte)  {
	if ms.expectSendSize < 1 {
		ms.expectSendSize = extractMysqlPayloadSize(bytes[:4])
		contents := bytes[4:]
		if ms.prepareInfo != nil && contents[0] == 0 {
			ms.prepareInfo.prepareStmtID = bytesToInt(contents[1:5])
		}
	}
}

func (ms *MysqlSession) mergeRanges()  {
	if len(ms.coverRanges) > 1 {
		newRange, newPkgRanges := mergeRanges(ms.coverRanges[0], ms.coverRanges[1:])
		tmpRanges := make([]*jigsaw, len(newPkgRanges)+1)
		tmpRanges[0] = newRange
		if len(newPkgRanges) > 0 {
			copy(tmpRanges[1:], newPkgRanges)
		}
		ms.coverRanges = tmpRanges
	}
}

func mergeRanges(currRange *jigsaw, pkgRanges []*jigsaw) (mergedRange *jigsaw, newPkgRanges []*jigsaw) {
	var nextRange *jigsaw
	newPkgRanges = make([]*jigsaw, 0, 4)

	if len(pkgRanges) < 1 {
		return currRange, newPkgRanges

	} else if len(pkgRanges) == 1 {
		nextRange = pkgRanges[0]

	} else {
		nextRange, newPkgRanges = mergeRanges(pkgRanges[0], pkgRanges[1:])
	}

	if currRange.e >= nextRange.b {
		mergedRange = &jigsaw{b: currRange.b, e: nextRange.e}

	} else {
		tmpRanges := make([]*jigsaw, len(newPkgRanges)+1)
		tmpRanges[0] = nextRange
		if len(newPkgRanges) > 0 {
			copy(tmpRanges[1:], newPkgRanges)
		}
		newPkgRanges = tmpRanges
		mergedRange = currRange
	}

	return
}

func (ms *MysqlSession) oneMysqlPackageFinish() bool {
	if int64(len(ms.cachedStmtBytes)) % MaxMysqlPacketLen == 0 {
		return true
	}

	return false
}

func (ms *MysqlSession) checkFinish() bool {
	if len(ms.coverRanges) != 1 {
		ranges := make([]string, 0, len(ms.coverRanges))
		for _, cr := range ms.coverRanges {
			log.Errorf("miss values: %s", string(ms.cachedStmtBytes[cr.b-ms.beginSeqID: cr.e-ms.beginSeqID]))

			ranges = append(ranges, fmt.Sprintf("[%d -- %d]", cr.b, cr.e))
		}


		log.Errorf("in session %s get invalid range: %s", *ms.connectionID, strings.Join(ranges, ", "))
		return false
	}

	firstRange := ms.coverRanges[0]
	if firstRange.e - firstRange.b != int64(len(ms.cachedStmtBytes)) {
		return false
	}

	return true
}

func (ms *MysqlSession) ReadFromClient(seqID int64, bytes []byte)  {
	contentSize := int64(len(bytes))

	if ms.expectReceiveSize == 0 || ms.oneMysqlPackageFinish() {
		ms.expectReceiveSize = extractMysqlPayloadSize(bytes[:4])
		ms.packageOffset = int64(len(ms.cachedStmtBytes))

		contents := bytes[4:]
		if contents[0] == ComStmtPrepare {
			ms.prepareInfo = &prepareInfo{}
		}

		contentSize = int64(len(contents))
		seqID += 4
		ms.beginSeqID = seqID
		newCache := make([]byte, ms.expectReceiveSize+len(ms.cachedStmtBytes))
		if len(ms.cachedStmtBytes) > 0 {
			copy(newCache[:len(ms.cachedStmtBytes)], ms.cachedStmtBytes)
		}

		if int64(ms.expectReceiveSize+len(ms.cachedStmtBytes)) > ms.packageOffset+int64(len(contents)) {
			copy(newCache[ms.packageOffset:ms.packageOffset+int64(len(contents))], contents)
			ms.cachedStmtBytes = newCache
		} else {
			log.Debugf("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXxxx")
			return
		}

	} else {
		if seqID < ms.beginSeqID {
			log.Debugf("in session %s get outdate package with Seq:%d", *ms.connectionID, seqID)
			return
		}

		seqOffset := seqID - ms.beginSeqID
		if ms.packageOffset+seqOffset+int64(len(bytes)) <= int64(ms.expectReceiveSize) {
			copy(ms.cachedStmtBytes[ms.packageOffset+seqOffset:ms.packageOffset+seqOffset+int64(len(bytes))], bytes)
		}
	}

	insertIdx := len(ms.coverRanges)
	for idx, cr := range ms.coverRanges {
		if seqID < cr.b {
			insertIdx = idx
			break
		}
	}

	cr := &jigsaw{b: seqID, e: seqID+contentSize}
	if len(ms.coverRanges) < 1 || insertIdx == len(ms.coverRanges) {
		ms.coverRanges = append(ms.coverRanges, cr)

	} else {
		newCoverRanges := make([]*jigsaw, len(ms.coverRanges)+1)
		copy(newCoverRanges[:insertIdx], ms.coverRanges[:insertIdx])
		newCoverRanges[insertIdx] = cr
		copy(newCoverRanges[insertIdx+1:], ms.coverRanges[insertIdx:])
		ms.coverRanges = newCoverRanges
	}

	ms.mergeRanges()
}

func (ms *MysqlSession) refreshWindowSize(readSize int)  {
	windowCounter := windowSizeCache[*ms.clientHost]
	if windowCounter == nil {
		windowCounter = newPackageWindowCounter()
		windowSizeCache[*ms.clientHost] = windowCounter
	}

	// windowCounter.refresh(readSize, ms.checkFinish())
	// ms.tcpWindowSize = windowCounter.suggestSize
}

func (ms *MysqlSession) GenerateQueryPiece() (qp model.QueryPiece) {
	defer func() {
		ms.cachedStmtBytes = nil
		ms.expectReceiveSize = 0
		ms.expectSendSize = 0
		ms.prepareInfo = nil
		ms.coverRanges = make([]*jigsaw, 0, 4)
		ms.lastSeq = -1
		ms.ackID = -1
		ms.sendSize = 0
	}()

	if len(ms.cachedStmtBytes) < 1 {
		return
	}

	// fmt.Printf("packageComplete in generate: %v\n", ms.packageComplete)
	if !ms.checkFinish() {
		log.Errorf("receive a not complete cover")
		return
	}

	var mqp *model.PooledMysqlQueryPiece
	var querySQLInBytes []byte
	switch ms.cachedStmtBytes[0] {
	case ComAuth:
		var userName, dbName string
		var err error
		userName, dbName, err = parseAuthInfo(ms.cachedStmtBytes)
		if err != nil {
			log.Errorf("parse auth info failed <-- %s", err.Error())
		}
		ms.visitUser = &userName
		ms.visitDB = &dbName

	case ComInitDB:
		newDBName := string(ms.cachedStmtBytes[1:])
		useSQL := fmt.Sprintf("use %s", newDBName)
		mqp = ms.composeQueryPiece()
		mqp.QuerySQL = &useSQL
		// update session database
		ms.visitDB = &newDBName

	case ComDropDB:
		dbName := string(ms.cachedStmtBytes[1:])
		dropSQL := fmt.Sprintf("drop database %s", dbName)
		mqp = ms.composeQueryPiece()
		mqp.QuerySQL = &dropSQL

	case ComCreateDB, ComQuery:
		mqp = ms.composeQueryPiece()
		querySQLInBytes = ms.cachedStmtBytes[1:]
		querySQL := hack.String(querySQLInBytes)
		mqp.QuerySQL = &querySQL

	case ComStmtPrepare:
		mqp = ms.composeQueryPiece()
		querySQLInBytes = ms.cachedStmtBytes[1:]
		querySQL := hack.String(querySQLInBytes)
		mqp.QuerySQL = &querySQL
		ms.cachedPrepareStmt[ms.prepareInfo.prepareStmtID] = &querySQL
		log.Debugf("prepare statement %s, get id:%d", querySQL, ms.prepareInfo.prepareStmtID)

	case ComStmtExecute:
		prepareStmtID := bytesToInt(ms.cachedStmtBytes[1:5])
		mqp = ms.composeQueryPiece()
		mqp.QuerySQL = ms.cachedPrepareStmt[prepareStmtID]
		log.Debugf("execute prepare statement:%d", prepareStmtID)

	case ComStmtClose:
		prepareStmtID := bytesToInt(ms.cachedStmtBytes[1:5])
		delete(ms.cachedPrepareStmt, prepareStmtID)
		log.Debugf("remove prepare statement:%d", prepareStmtID)

	default:
		return
	}

	if strictMode && mqp != nil && mqp.VisitUser == nil {
		user, db, err := querySessionInfo(ms.serverPort, mqp.SessionID)
		if err != nil {
			log.Errorf("query user and db from mysql failed <-- %s", err.Error())
		} else {
			mqp.VisitUser = user
			mqp.VisitDB = db
		}
	}

	return filterQueryPieceBySQL(mqp, querySQLInBytes)
}

func filterQueryPieceBySQL(mqp *model.PooledMysqlQueryPiece, querySQL []byte) (model.QueryPiece) {
	if mqp == nil || querySQL == nil {
		return nil

	} else if (uselessSQLPattern.Match(querySQL)) {
		return nil
	}

	if ddlPatern.Match(querySQL) {
		mqp.SetNeedSyncSend(true)
	}

	return mqp
}

func (ms *MysqlSession) composeQueryPiece() (mqp *model.PooledMysqlQueryPiece) {
	return model.NewPooledMysqlQueryPiece(
		ms.connectionID, ms.visitUser, ms.visitDB, ms.clientHost, ms.serverIP, ms.serverPort, ms.stmtBeginTime)
}
