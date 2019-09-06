package mysql

import (
	"fmt"
	"github.com/zr-hebo/sniffer-agent/communicator"

	// "strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/pingcap/tidb/util/hack"
	"github.com/zr-hebo/sniffer-agent/model"
)

type MysqlSession struct {
	connectionID             *string
	visitUser                *string
	visitDB                  *string
	clientHost               *string
	clientPort               int
	serverIP                 *string
	serverPort               int
	stmtBeginTime            int64
	packageOffset            int64
	expectReceiveSize        int
	// coverRanges              []*receiveRange
	// coverRange               *receiveRange
	beginSeqID  int64
	expectSeqID int64

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
	sessionKey *string, clientIP *string, clientPort int, serverIP *string, serverPort int,
	receiver chan model.QueryPiece) (ms *MysqlSession) {
	ms = &MysqlSession{
		connectionID:       sessionKey,
		clientHost:         clientIP,
		clientPort:         clientPort,
		serverIP:           serverIP,
		serverPort:         serverPort,
		stmtBeginTime:      time.Now().UnixNano() / millSecondUnit,
		cachedPrepareStmt:  make(map[int][]byte, 8),
		queryPieceReceiver: receiver,
		closeConn:          make(chan bool, 1),
		expectReceiveSize:  -1,
		expectSeqID:        -1,
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
		ms.readFromServer(newPkt.Payload)
		qp := ms.GenerateQueryPiece()
		if qp != nil {
			ms.queryPieceReceiver <- qp
		}
	}
}

func (ms *MysqlSession) resetBeginTime() {
	ms.stmtBeginTime = time.Now().UnixNano() / millSecondUnit
}

func (ms *MysqlSession) readFromServer(bytes []byte) {
	if ms.expectSendSize < 1 {
		ms.expectSendSize = extractMysqlPayloadSize(bytes[:4])
		contents := bytes[4:]
		if ms.prepareInfo != nil && contents[0] == 0 {
			ms.prepareInfo.prepareStmtID = bytesToInt(contents[1:5])
		}
	}
}

func (ms *MysqlSession) oneMysqlPackageFinish() bool {
	if int64(len(ms.cachedStmtBytes))%MaxMysqlPacketLen == 0 {
		return true
	}

	return false
}

func (ms *MysqlSession) checkFinish() bool {
	if ms.beginSeqID != -1 && ms.expectReceiveSize == 0 {
		return true
	}

	return false
}

func (ms *MysqlSession) clear() {
	ms.cachedStmtBytes = nil
	ms.expectReceiveSize = -1
	ms.expectSendSize = -1
	ms.prepareInfo = nil
	ms.beginSeqID = -1
	ms.expectSeqID = -1
	ms.ignoreAckID = -1
	ms.sendSize = 0
}

func (ms *MysqlSession) readFromClient(seqID int64, bytes []byte) {
	contentSize := int64(len(bytes))

	if ms.expectReceiveSize == -1 || ms.oneMysqlPackageFinish() {
		ms.expectReceiveSize = extractMysqlPayloadSize(bytes[:4])
		// ms.packageOffset = int64(len(ms.cachedStmtBytes))

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

		if int64(ms.expectReceiveSize+len(ms.cachedStmtBytes)) >= ms.packageOffset+int64(len(contents)) {
			copy(newCache[ms.packageOffset:ms.packageOffset+int64(len(contents))], contents)
			ms.cachedStmtBytes = newCache
		}

	} else {
		if ms.beginSeqID == -1 {
			log.Warnf("cover range is empty")
			return
		}

		if seqID < ms.beginSeqID {
			// out date packet
			log.Debugf("in session %s get outdate package with Seq:%d, beginSeq:%d",
				*ms.connectionID, seqID, ms.beginSeqID)
			return

		} else if seqID + int64(len(bytes)) <= ms.beginSeqID {
			// repeat packet
			log.Debugf("receive repeat packet")
			return

		} else if seqID > ms.expectSeqID {
			// discontinuous packet
			log.Debugf("receive discontinuous packet")
			ms.clear()
			return
		}

		seqOffset := seqID - ms.beginSeqID + ms.packageOffset
		if seqOffset+int64(len(bytes)) > int64(len(ms.cachedStmtBytes)) {
			// is not a normal mysql packet
			log.Debugf("receive an unexpect packet")
			ms.clear()
			return
		}

		// add byte to stmt cache
		copy(ms.cachedStmtBytes[ms.packageOffset+seqOffset:ms.packageOffset+seqOffset+contentSize], bytes)
	}

	ms.expectReceiveSize = ms.expectReceiveSize - int(contentSize)
	ms.expectSeqID = seqID + contentSize
}

func (ms *MysqlSession) GenerateQueryPiece() (qp model.QueryPiece) {
	defer ms.clear()

	if len(ms.cachedStmtBytes) < 1 {
		return
	}

	if !ms.checkFinish() {
		log.Debugf("receive a not complete cover")
		return
	}

	if len(ms.cachedStmtBytes) > maxSQLLen {
		log.Warn("sql in cache is too long, ignore it")
		return
	}

	var mqp *model.PooledMysqlQueryPiece
	var querySQLInBytes []byte
	if ms.cachedStmtBytes[0] > 32 {
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
			querySQLInBytes = ms.cachedStmtBytes[1:]
			querySQL := hack.String(querySQLInBytes)
			mqp.QuerySQL = &querySQL
			ms.cachedPrepareStmt[ms.prepareInfo.prepareStmtID] = querySQLInBytes
			log.Debugf("prepare statement %s, get id:%d", querySQL, ms.prepareInfo.prepareStmtID)

		case ComStmtExecute:
			prepareStmtID := bytesToInt(ms.cachedStmtBytes[1:5])
			mqp = ms.composeQueryPiece()
			querySQLInBytes = ms.cachedPrepareStmt[prepareStmtID]
			querySQL := hack.String(querySQLInBytes)
			mqp.QuerySQL = &querySQL
			log.Debugf("execute prepare statement:%d", prepareStmtID)

		case ComStmtClose:
			prepareStmtID := bytesToInt(ms.cachedStmtBytes[1:5])
			delete(ms.cachedPrepareStmt, prepareStmtID)
			log.Debugf("remove prepare statement:%d", prepareStmtID)

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
		ms.connectionID, ms.clientHost, ms.visitUser, ms.visitDB, ms.clientHost, ms.serverIP,
		ms.clientPort, ms.serverPort, communicator.GetConfig(communicator.THROW_PACKET_RATE).(float64),
		ms.stmtBeginTime)
}
