package mysql

import (
	"fmt"
	"github.com/siddontang/go/hack"
	"time"

	"github.com/zr-hebo/sniffer-agent/model"
	log "github.com/sirupsen/logrus"
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
	expectReceiveSize int
	packageBaseSize   int
	packageComplete   bool
	expectSendSize    int
	prepareInfo       *prepareInfo
	sizeCount map[int]int64
	cachedPrepareStmt map[int]*string
	tcpCache          []byte
	cachedStmtBytes   []byte
}

type prepareInfo struct {
	prepareStmtID int
}

const (
	defaultCacheSize  = 1<<16
	maxBeyondCount = 3
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
	ms.tcpCache = make([]byte, 0, defaultCacheSize)
	ms.cachedStmtBytes = make([]byte, 0, defaultCacheSize)
	ms.packageBaseSize = 512
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
		fmt.Printf("Init ms.expectSendSize: %v\n", ms.expectSendSize)
		ms.expectSendSize = ms.expectSendSize - len(contents)

	} else {
		ms.expectSendSize = ms.expectSendSize - len(bytes)
	}
}

func (ms *MysqlSession) ReadFromClient(bytes []byte)  {
	if ms.expectReceiveSize < 1 {
		ms.expectReceiveSize = extractMysqlPayloadSize(bytes[:4])
		contents := bytes[4:]
		if contents[0] == ComStmtPrepare {
			ms.prepareInfo = &prepareInfo{}
		}

		ms.tcpCache = append(ms.tcpCache, contents...)
		ms.expectReceiveSize = ms.expectReceiveSize - len(contents)

	} else {
		ms.tcpCache = append(ms.tcpCache, bytes...)
		ms.expectReceiveSize = ms.expectReceiveSize - len(bytes)
	}

	readSize := len(bytes)
	readTail := readSize % ms.packageBaseSize
	if readTail != 0 {
		if ms.expectReceiveSize == 0 {
			ms.cachedStmtBytes = append(ms.cachedStmtBytes, ms.tcpCache...)
			ms.tcpCache = ms.tcpCache[0:0]
			ms.packageComplete = true

		} else {
			ms.packageComplete = false
		}

	} else if readTail == 0 && ms.expectReceiveSize == 0 {
		ms.packageComplete = true
	}

	miniMatchSize := 1 << 16
	for size := range ms.sizeCount  {
		if readSize % size == 0 && miniMatchSize > size {
			miniMatchSize = size
		}
	}
	if miniMatchSize < 1 << 16 {
		ms.sizeCount[miniMatchSize] = ms.sizeCount[miniMatchSize] + 1
	} else if (ms.expectReceiveSize != 0) {
		ms.sizeCount[readSize] = 1
	}



	mostFrequentSize := ms.packageBaseSize
	miniSize := ms.packageBaseSize
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

	ms.packageBaseSize = mostFrequentSize

	// fmt.Printf("read %v bytes: %v\n", len(bytes), string(bytes))
	fmt.Printf("ms.expectReceiveSize: %v\n", ms.expectReceiveSize)
	fmt.Printf("ms.sizeCount: %#v\n", ms.sizeCount)
	fmt.Printf("len(ms.tcpCache): %#v\n", len(ms.tcpCache))
	fmt.Printf("packageComplete in read: %v\n", ms.packageComplete)
	log.Infof("ms.packageBaseSize: %v", ms.packageBaseSize)
}

func (ms *MysqlSession) ReadOnePackageFinish() bool {
	if len(ms.tcpCache) == MaxPayloadLen {
		return true
	}

	return false
}

func (ms *MysqlSession) ReadAllPackageFinish() bool {
	// fmt.Printf("len(ms.tcpCache): %v\n", len(ms.tcpCache))
	// fmt.Printf("ms.expectReceiveSize: %v\n", ms.expectReceiveSize)

	if len(ms.tcpCache) < MaxPayloadLen {
		return true
	}

	return false
}

func (ms *MysqlSession) ResetCache() {
	ms.cachedStmtBytes = append(ms.cachedStmtBytes, ms.tcpCache...)
	ms.tcpCache = ms.tcpCache[0:0]
}

func (ms *MysqlSession) GenerateQueryPiece() (qp model.QueryPiece) {
	defer func() {
		ms.tcpCache = ms.tcpCache[0:0]
		ms.cachedStmtBytes = ms.cachedStmtBytes[0:0]
		ms.expectReceiveSize = 0
		ms.expectSendSize = 0
		ms.prepareInfo = nil
		ms.packageComplete = false
	}()

	if len(ms.cachedStmtBytes) < 1 && len(ms.tcpCache) < 1 {
		return
	}

	// fmt.Printf("packageComplete in generate: %v\n", ms.packageComplete)
	if !ms.packageComplete {
		return
	}

	var mqp *model.PooledMysqlQueryPiece
	var querySQLInBytes []byte
	ms.cachedStmtBytes = append(ms.cachedStmtBytes, ms.tcpCache...)
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
		querySQLInBytes = make([]byte, len(ms.cachedStmtBytes[1:]))
		copy(querySQLInBytes, ms.cachedStmtBytes[1:])
		querySQL := hack.String(querySQLInBytes)
		mqp.QuerySQL = &querySQL

	case ComStmtPrepare:
		mqp = ms.composeQueryPiece()
		querySQLInBytes = make([]byte, len(ms.cachedStmtBytes[1:]))
		copy(querySQLInBytes, ms.cachedStmtBytes[1:])
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
