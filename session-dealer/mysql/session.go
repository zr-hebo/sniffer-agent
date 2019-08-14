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
	expectSize        int
	prepareInfo       *prepareInfo
	cachedPrepareStmt map[int]*string
	tcpCache          []byte
	cachedStmtBytes   []byte
}

type prepareInfo struct {
	prepareStmtID int
}

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
	return
}

func (ms *MysqlSession) ResetBeginTime()  {
	ms.stmtBeginTime = time.Now().UnixNano() / millSecondUnit
}

func (ms *MysqlSession) ReadFromServer(bytes []byte)  {
	if ms.expectSize < 1 {
		ms.expectSize = extractMysqlPayloadSize(bytes)
		contents := bytes[4:]
		if ms.prepareInfo != nil && contents[0] == 0 {
			ms.prepareInfo.prepareStmtID = bytesToInt(contents[1:5])
		}
		ms.expectSize = ms.expectSize - len(contents)

	} else {
		ms.expectSize = ms.expectSize - len(bytes)
	}
}

func (ms *MysqlSession) ReadFromClient(bytes []byte)  {
	if ms.expectSize < 1 {
		ms.expectSize = extractMysqlPayloadSize(bytes)
		contents := bytes[4:]
		if contents[0] == ComStmtPrepare {
			ms.prepareInfo = &prepareInfo{}
		}

		ms.expectSize = ms.expectSize - len(contents)
		ms.tcpCache = append(ms.tcpCache, contents...)

	} else {
		ms.expectSize = ms.expectSize - len(bytes)
		ms.tcpCache = append(ms.tcpCache, bytes...)
		if len(ms.tcpCache) == MaxPayloadLen {
			ms.cachedStmtBytes = append(ms.cachedStmtBytes, ms.tcpCache...)
			ms.tcpCache = ms.tcpCache[:0]
			ms.expectSize = 0
		}
	}
}

func (ms *MysqlSession) GenerateQueryPiece() (qp model.QueryPiece) {
	if len(ms.cachedStmtBytes) < 1 && len(ms.tcpCache) < 1 {
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
			return
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

	case ComCreateDB:
	case ComDropDB:
	case ComQuery:
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

	ms.tcpCache = ms.tcpCache[:0]
	ms.cachedStmtBytes = ms.cachedStmtBytes[:0]
	ms.expectSize = 0
	ms.prepareInfo = nil
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
