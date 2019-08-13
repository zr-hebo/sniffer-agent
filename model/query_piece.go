package model

import (
	"encoding/json"
	"time"

	"github.com/pingcap/tidb/util/hack"
)

type QueryPiece interface {
	String() *string
	Bytes() []byte
	GetSQL() *string
	NeedSyncSend() bool
	Recovery()
}

// MysqlQueryPiece 查询信息
type MysqlQueryPiece struct {
	SessionID    *string `json:"sid"`
	ClientHost   *string `json:"-"`
	SyncSend     bool    `json:"-"`
	ServerIP     *string `json:"sip"`
	ServerPort   int     `json:"sport"`
	VisitUser    *string `json:"user"`
	VisitDB      *string `json:"db"`
	QuerySQL     *string `json:"sql"`
	BeginTime    string  `json:"bt"`
	CostTimeInMS int64   `json:"cms"`
}

type PooledMysqlQueryPiece struct {
	MysqlQueryPiece
	recoverPool *mysqlQueryPiecePool
}

const (
	datetimeFormat = "2006-01-02 15:04:05"
	millSecondUnit = int64(time.Millisecond)
)

var (
	mqpp = NewMysqlQueryPiecePool()
)

func NewPooledMysqlQueryPiece(
	sessionID, visitUser, visitDB, clientHost, serverIP *string, serverPort int, stmtBeginTime int64) (
	mqp *PooledMysqlQueryPiece) {
	mqp = mqpp.Dequeue()
	if mqp == nil {
		mqp = &PooledMysqlQueryPiece{
			MysqlQueryPiece: MysqlQueryPiece{},
		}
	}

	nowInMS := time.Now().UnixNano() / millSecondUnit
	mqp.SessionID = sessionID
	mqp.ClientHost = clientHost
	mqp.ServerIP = serverIP
	mqp.ServerPort = serverPort
	mqp.VisitUser = visitUser
	mqp.VisitDB = visitDB
	mqp.SyncSend = false
	mqp.BeginTime = time.Unix(stmtBeginTime/1000, 0).Format(datetimeFormat)
	mqp.CostTimeInMS = nowInMS - stmtBeginTime
	mqp.recoverPool = mqpp

	return
}

func (qp *MysqlQueryPiece) String() (*string) {
	content := qp.Bytes()
	contentStr := hack.String(content)
	return &contentStr
}

func (qp *MysqlQueryPiece) Bytes() (bytes []byte) {
	content, err := json.Marshal(qp)
	if err != nil {
		return []byte(err.Error())
	}

	return content
}

func (qp *MysqlQueryPiece) GetSQL() (str *string) {
	return qp.QuerySQL
}

func (qp *MysqlQueryPiece) NeedSyncSend() (bool) {
	return qp.SyncSend
}

func (qp *MysqlQueryPiece) SetNeedSyncSend(syncSend bool) {
	qp.SyncSend = syncSend
}

func (pmqp *PooledMysqlQueryPiece) Recovery() {
	pmqp.recoverPool.Enqueue(pmqp)
}
