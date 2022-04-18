package mysql

import (
	"fmt"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/pingcap/tidb/util/hack"
	"github.com/zr-hebo/sniffer-agent/communicator"
	"github.com/zr-hebo/sniffer-agent/model"
)

type MysqlSession struct {
	connectionID      *string
	visitUser         *string
	visitDB           *string
	clientIP          *string
	clientPort        int
	serverIP          *string
	serverPort        int
	stmtBeginTimeNano int64
	// packageOffset            int64
	beginSeqID               int64
	endSeqID                 int64
	coverRanges              *coverRanges
	expectReceiveSize        int
	expectSendSize           int
	prepareInfo              *prepareInfo
	cachedPrepareStmt        map[int][]byte
	cachedStmtBytes          []byte
	computeWindowSizeCounter int

	tcpPacketCache []*model.TCPPacket

	queryPieceReceiver chan model.QueryPiece
	closeConn          chan bool
	pkgCacheLock       sync.Mutex

	ignoreAckID int64
	sendSize    int64
}

type prepareInfo struct {
	prepareStmtID int
}

func NewMysqlSession(
	sessionKey, clientIP *string, clientPort int, serverIP *string, serverPort int,
	receiver chan model.QueryPiece) (ms *MysqlSession) {
	ms = &MysqlSession{
		connectionID:       sessionKey,
		clientIP:           clientIP,
		clientPort:         clientPort,
		serverIP:           serverIP,
		serverPort:         serverPort,
		stmtBeginTimeNano:  time.Now().UnixNano(),
		cachedPrepareStmt:  make(map[int][]byte, 8),
		queryPieceReceiver: receiver,
		closeConn:          make(chan bool, 1),
		expectReceiveSize:  -1,
		coverRanges:        NewCoverRanges(),
		ignoreAckID:        -1,
		sendSize:           0,
		pkgCacheLock:       sync.Mutex{},
	}

	return
}

func (ms *MysqlSession) ReceiveTCPPacket(newPkt *model.TCPPacket) {
	if newPkt == nil {
		return
	}

	if !newPkt.ToServer && ms.ignoreAckID == newPkt.Seq {
		// ignore to response to client data
		ms.ignoreAckID = ms.ignoreAckID + int64(len(newPkt.Payload))
		return

	} else if !newPkt.ToServer {
		ms.ignoreAckID = newPkt.Seq + int64(len(newPkt.Payload))
	}

	if newPkt.ToServer {
		ms.resetBeginTime()
		ms.readFromClient(newPkt.Seq, newPkt.Payload)

	} else {
		ms.readFromServer(newPkt.Seq, newPkt.Payload)
		qp := ms.GenerateQueryPiece()
		if qp != nil {
			ms.queryPieceReceiver <- qp
		}
	}
}

func (ms *MysqlSession) resetBeginTime() {
	ms.stmtBeginTimeNano = time.Now().UnixNano()
}

func (ms *MysqlSession) readFromServer(respSeq int64, bytes []byte) {
	if ms.expectSendSize < 1 && len(bytes) > 4 {
		ms.expectSendSize = extractMysqlPayloadSize(bytes[:4])
		contents := bytes[4:]
		if ms.prepareInfo != nil && contents[0] == 0 {
			ms.prepareInfo.prepareStmtID = bytesToInt(contents[1:5])
		}
	}

	if ms.coverRanges.head.next == nil || ms.coverRanges.head.next.end != respSeq {
		ms.clear()
	}
}

func (ms *MysqlSession) checkFinish() bool {
	if ms.coverRanges.head == nil || ms.coverRanges.head.next == nil {
		return false
	}

	checkNode := ms.coverRanges.head.next
	if checkNode.end-checkNode.begin == int64(len(ms.cachedStmtBytes)) {
		return true
	}

	return false
}

func (ms *MysqlSession) Close() {
	ms.clear()
}

func (ms *MysqlSession) clear() {
	localStmtCache.Enqueue(ms.cachedStmtBytes)
	ms.cachedStmtBytes = nil
	ms.expectReceiveSize = -1
	ms.expectSendSize = -1
	ms.prepareInfo = nil
	ms.beginSeqID = -1
	ms.endSeqID = -1
	ms.ignoreAckID = -1
	ms.sendSize = 0
	ms.coverRanges.clear()
}

func (ms *MysqlSession) readFromClient(seqID int64, bytes []byte) {
	contentSize := int64(len(bytes))

	if ms.expectReceiveSize == -1 {
		// ignore invalid head package
		if len(bytes) <= 4 {
			return
		}

		ms.expectReceiveSize = extractMysqlPayloadSize(bytes[:4])
		// ignore too big mysql packet
		if ms.expectReceiveSize >= MaxMySQLPacketLen {
			log.Infof("expect receive size is bigger than max deal size: %d", MaxMySQLPacketLen)
			return
		}

		contents := bytes[4:]
		// add prepare info
		if contents[0] == ComStmtPrepare {
			ms.prepareInfo = &prepareInfo{}
		}

		contentSize = int64(len(contents))
		seqID += 4
		ms.beginSeqID = seqID
		ms.endSeqID = seqID

		if int64(ms.expectReceiveSize) < int64(len(contents)) {
			log.Warning("receive invalid mysql packet")
			return
		}

		newCache := localStmtCache.DequeueWithInit(ms.expectReceiveSize)
		copy(newCache[:len(contents)], contents)
		ms.cachedStmtBytes = newCache

	} else {
		// ignore too big mysql packet
		if ms.expectReceiveSize >= MaxMySQLPacketLen {
			return
		}

		if ms.beginSeqID == -1 {
			log.Info("cover range is empty")
			return
		}

		if seqID < ms.beginSeqID {
			// out date packet
			log.Infof("in session %s get outdate package with Seq:%d, beginSeq:%d",
				*ms.connectionID, seqID, ms.beginSeqID)
			return
		}

		seqOffset := seqID - ms.beginSeqID
		if seqOffset+contentSize > int64(len(ms.cachedStmtBytes)) {
			// not in a normal mysql packet
			log.Info("receive an unexpect packet")
			ms.clear()
			return
		}

		// add byte to stmt cache
		copy(ms.cachedStmtBytes[seqOffset:seqOffset+contentSize], bytes)
	}

	ms.coverRanges.addRange(coverRangePool.NewCoverage(seqID, seqID+contentSize))
	// ms.expectReceiveSize = ms.expectReceiveSize - int(contentSize)
}

func IsAuth(val byte) bool {
	return val > 32
}

func (ms *MysqlSession) GenerateQueryPiece() (qp model.QueryPiece) {
	defer ms.clear()

	if len(ms.cachedStmtBytes) < 1 {
		return
	}

	if !ms.checkFinish() {
		log.Warning("receive a not complete cover")
		return
	}

	if len(ms.cachedStmtBytes) > maxSQLLen {
		log.Warning("sql in cache is too long, ignore it")
		return
	}

	var mqp *model.PooledMysqlQueryPiece
	var querySQLInBytes []byte
	if IsAuth(ms.cachedStmtBytes[0]) {
		userName, dbName, err := parseAuthInfo(ms.cachedStmtBytes)
		if err != nil {
			log.Errorf("parse auth info failed <-- %s", err.Error())
			return
		}
		ms.visitUser = &userName
		ms.visitDB = &dbName

	} else {
		switch ms.cachedStmtBytes[0] {
		case ComInitDB:
			newDBName := string(ms.cachedStmtBytes[1:])
			useSQL := fmt.Sprintf("use %s", newDBName)
			querySQLInBytes = hack.Slice(useSQL)
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
			querySQLInBytes = make([]byte, len(ms.cachedStmtBytes[1:]))
			copy(querySQLInBytes, ms.cachedStmtBytes[1:])
			querySQL := hack.String(querySQLInBytes)
			mqp.QuerySQL = &querySQL
			ms.cachedPrepareStmt[ms.prepareInfo.prepareStmtID] = querySQLInBytes
			log.Infof("prepare statement %s, get id:%d", querySQL, ms.prepareInfo.prepareStmtID)

		case ComStmtExecute:
			prepareStmtID := bytesToInt(ms.cachedStmtBytes[1:5])
			mqp = ms.composeQueryPiece()
			var ok bool
			querySQLInBytes, ok = ms.cachedPrepareStmt[prepareStmtID]
			if !ok {
				querySQLInBytes = PrepareStatement
			}
			querySQL := hack.String(querySQLInBytes)
			mqp.QuerySQL = &querySQL

			// log.Debugf("execute prepare statement:%d", prepareStmtID)

		case ComStmtClose:
			prepareStmtID := bytesToInt(ms.cachedStmtBytes[1:5])
			delete(ms.cachedPrepareStmt, prepareStmtID)
			log.Infof("remove prepare statement:%d", prepareStmtID)

		default:
			return
		}
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

	mqp = filterQueryPieceBySQL(mqp, querySQLInBytes)
	if mqp == nil {
		return nil
	}

	communicator.ReceiveExecTime(ms.stmtBeginTimeNano)
	return mqp
}

func filterQueryPieceBySQL(mqp *model.PooledMysqlQueryPiece, querySQL []byte) *model.PooledMysqlQueryPiece {
	if mqp == nil || querySQL == nil {
		return nil

	} else if uselessSQLPattern.Match(querySQL) {
		return nil
	}

	if ddlPatern.Match(querySQL) {
		mqp.SetNeedSyncSend(true)
	}

	return mqp
}

func (ms *MysqlSession) composeQueryPiece() (mqp *model.PooledMysqlQueryPiece) {
	clientIP := ms.clientIP
	clientPort := ms.clientPort
	return model.NewPooledMysqlQueryPiece(
		ms.connectionID, clientIP, ms.visitUser, ms.visitDB, ms.serverIP,
		clientPort, ms.serverPort, communicator.GetMysqlCapturePacketRate(), ms.stmtBeginTimeNano)
}
