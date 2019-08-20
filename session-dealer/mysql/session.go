package mysql

import (
	"fmt"
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
	tcpWindowSize     int
	expectSendSize    int
	prepareInfo       *prepareInfo
	sizeCount         map[int]int64
	cachedPrepareStmt map[int]*string
	cachedStmtBytes   []byte
	computeWindowSizeCounter int
}

type prepareInfo struct {
	prepareStmtID int
}

const (
	defaultCacheSize  = 1 << 16
	maxIPPackageSize = 1 << 16
)

func NewMysqlSession(sessionKey *string, clientIP *string, clientPort int, serverIP *string, serverPort int) (ms *MysqlSession) {
	ms = &MysqlSession{
		connectionID:      sessionKey,
		clientHost:        clientIP,
		clientPort:        clientPort,
		serverIP:          serverIP,
		serverPort:        serverPort,
		stmtBeginTime:     time.Now().UnixNano() / millSecondUnit,
		cachedPrepareStmt: make(map[int]*string, 8),
	}
	ms.tcpWindowSize = 512
	ms.coverRanges = make([]*jigsaw, 0, 4)
	ms.sizeCount = make(map[int]int64)

	return
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
		newPkgRanges = append(newPkgRanges, newRange)
		ms.coverRanges = newPkgRanges
	}
}

func mergeRanges(currRange *jigsaw, pkgRanges []*jigsaw) (mergedRange *jigsaw, newPkgRanges []*jigsaw) {
	var nextRange *jigsaw
	if len(pkgRanges) < 1 {
		return currRange, make([]*jigsaw, 0)

	} else if len(pkgRanges) == 1 {
		nextRange = pkgRanges[0]
		newPkgRanges = make([]*jigsaw, 0, 4)

	} else {
		nextRange, newPkgRanges = mergeRanges(pkgRanges[0], pkgRanges[1:])
	}

	if currRange.e >= nextRange.b {
		mergedRange = &jigsaw{b: currRange.b, e: nextRange.e}

	} else {
		newPkgRanges = append(newPkgRanges, nextRange)
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
		return true
	}

	firstRange := ms.coverRanges[0]
	if firstRange.e - firstRange.b != int64(len(ms.cachedStmtBytes)) {
		return false
	}

	return true
}

func (ms *MysqlSession) ReadFromClient(seqID int64, bytes []byte)  {
	readSize := len(bytes)
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
		copy(newCache[ms.packageOffset:ms.packageOffset+int64(len(contents))], contents)
		ms.cachedStmtBytes = newCache

	} else {
		if seqID < ms.beginSeqID {
			log.Debugf("outdate package with Seq:%d", seqID)
			return
		}

		seqOffset := seqID - ms.beginSeqID
		if ms.packageOffset+seqOffset+int64(len(bytes)) <= int64(ms.expectReceiveSize) {
			copy(ms.cachedStmtBytes[ms.packageOffset+seqOffset:ms.packageOffset+seqOffset+int64(len(bytes))], bytes)
		}
	}

	ms.refreshWindowSize(readSize)

	insertIdx := len(ms.coverRanges)
	for idx, cr := range ms.coverRanges {
		if seqID < cr.b {
			insertIdx = idx
			break
		}
	}

	cr := &jigsaw{b: seqID, e: seqID+int64(contentSize)}
	if insertIdx == len(ms.coverRanges) - 1 {
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
	if ms.computeWindowSizeCounter > 5000 {
		return
	}

	log.Debugf("sizeCount: %#v", ms.sizeCount)

	ms.computeWindowSizeCounter += 1
	miniMatchSize := maxIPPackageSize
	for size := range ms.sizeCount  {
		if readSize % size == 0 && miniMatchSize > size {
			miniMatchSize = size
		}
	}
	if miniMatchSize < maxIPPackageSize {
		ms.sizeCount[miniMatchSize] = ms.sizeCount[miniMatchSize] + 1
	} else if ms.checkFinish() {
		ms.sizeCount[readSize] = 1
	}

	mostFrequentSize := ms.tcpWindowSize
	miniSize := ms.tcpWindowSize
	mostFrequentCount := int64(0)
	for size, count := range ms.sizeCount  {
		if count > mostFrequentCount {
			mostFrequentSize = size
			mostFrequentCount = count
		}

		if miniSize > size {
			miniSize = size
		}
	}

	ms.tcpWindowSize = mostFrequentSize
}


func (ms *MysqlSession) GenerateQueryPiece() (qp model.QueryPiece) {
	defer func() {
		// ms.tcpCache = ms.tcpCache[0:0]
		ms.cachedStmtBytes = nil
		ms.expectReceiveSize = 0
		ms.expectSendSize = 0
		ms.prepareInfo = nil
		ms.coverRanges = make([]*jigsaw, 0, 4)
		// ms.packageComplete = false
	}()

	if len(ms.cachedStmtBytes) < 1 {
		return
	}

	// fmt.Printf("packageComplete in generate: %v\n", ms.packageComplete)
	if !ms.checkFinish() {
		log.Errorf("is not a complete cover")
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

	// log.Debug(mqp.String())
	return mqp
}

func (ms *MysqlSession) composeQueryPiece() (mqp *model.PooledMysqlQueryPiece) {
	return model.NewPooledMysqlQueryPiece(
		ms.connectionID, ms.visitUser, ms.visitDB, ms.clientHost, ms.serverIP, ms.serverPort, ms.stmtBeginTime)
}
