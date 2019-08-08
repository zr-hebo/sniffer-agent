package mysql

import (
	"fmt"
	"time"

	"github.com/zr-hebo/sniffer-agent/model"
	log "github.com/sirupsen/logrus"
)

type MysqlSession struct {
	connectionID    string
	visitUser       *string
	visitDB         *string
	clientHost      string
	clientPort      int
	serverIP      string
	serverPort      int
	beginTime       int64
	expectSize       int
	prepareInfo *prepareInfo
	cachedPrepareStmt map[int]*string
	tcpCache []byte
	cachedStmtBytes []byte
}

type prepareInfo struct {
	prepareStmtID int
}

func NewMysqlSession(sessionKey string, clientIP string, clientPort int, serverIP string, serverPort int) (ms *MysqlSession) {
	ms = &MysqlSession{
		connectionID: sessionKey,
		clientHost: clientIP,
		clientPort: clientPort,
		serverIP: serverIP,
		serverPort: serverPort,
		beginTime: time.Now().UnixNano() / int64(time.Millisecond),
		cachedPrepareStmt: make(map[int]*string),
	}
	return
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

	var mqp *model.MysqlQueryPiece = nil
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
		querySQL := string(ms.cachedStmtBytes[1:])
		mqp.QuerySQL = &querySQL

	case ComStmtPrepare:
		mqp = ms.composeQueryPiece()
		querySQL := string(ms.cachedStmtBytes[1:])
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
	return filterQueryPieceBySQL(mqp)
}

func filterQueryPieceBySQL(mqp *model.MysqlQueryPiece) (model.QueryPiece) {
	if mqp == nil || mqp.QuerySQL == nil {
		return nil

	} else if (uselessSQLPattern.MatchString(*mqp.QuerySQL)) {
		return nil
	}

	return mqp
}

func (ms *MysqlSession) composeQueryPiece() (mqp *model.MysqlQueryPiece) {
	nowInMS := time.Now().UnixNano() / int64(time.Millisecond)
	mqp = &model.MysqlQueryPiece{
		SessionID:    ms.connectionID,
		ClientHost:   ms.clientHost,
		ServerIP:     ms.serverIP,
		ServerPort:   ms.serverPort,
		VisitUser:    ms.visitUser,
		VisitDB:      ms.visitDB,
		BeginTime:    time.Unix(ms.beginTime/1000, 0).Format(datetimeFormat),
		CostTimeInMS: nowInMS - ms.beginTime,
	}
	return mqp
}
