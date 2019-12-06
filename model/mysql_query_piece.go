package model

import (
	"github.com/pingcap/tidb/util/hack"
	"time"
)

// MysqlQueryPiece 查询信息
type MysqlQueryPiece struct {
	BaseQueryPiece

	SessionID    *string `json:"-"`
	ClientHost   *string `json:"cip"`
	ClientPort   int     `json:"cport"`

	VisitUser    *string `json:"user"`
	VisitDB      *string `json:"db"`
	QuerySQL     *string `json:"sql"`
	CostTimeInMS int64   `json:"cms"`
}

type PooledMysqlQueryPiece struct {
	MysqlQueryPiece
	recoverPool *mysqlQueryPiecePool
	sliceBufferPool *sliceBufferPool
}

func NewPooledMysqlQueryPiece(
	sessionID, clientIP, visitUser, visitDB, serverIP *string,
	clientPort, serverPort int, throwPacketRate float64, stmtBeginTime int64) (
	mqp *PooledMysqlQueryPiece) {
	mqp = mqpp.Dequeue()

	nowInMS := time.Now().UnixNano() / millSecondUnit
	mqp.SessionID = sessionID
	mqp.ClientHost = clientIP
	mqp.ClientPort = clientPort
	mqp.ServerIP = serverIP
	mqp.ServerPort = serverPort
	mqp.VisitUser = visitUser
	mqp.VisitDB = visitDB
	mqp.SyncSend = false
	mqp.CapturePacketRate = throwPacketRate
	mqp.BeginTime = stmtBeginTime
	mqp.CostTimeInMS = nowInMS - stmtBeginTime
	mqp.recoverPool = mqpp
	mqp.sliceBufferPool = localSliceBufferPool

	return
}

func (mqp *MysqlQueryPiece) String() (*string) {
	content := mqp.Bytes()
	contentStr := hack.String(content)
	return &contentStr
}

func (mqp *MysqlQueryPiece) Bytes() (content []byte) {
	// content, err := json.Marshal(mqp)
	if len(mqp.jsonContent) > 0 {
		return mqp.jsonContent
	}

	mqp.GenerateJsonBytes()
	return mqp.jsonContent
}

func (mqp *MysqlQueryPiece) GenerateJsonBytes() {
	mqp.jsonContent = marsharQueryPieceMonopolize(mqp)
	return
}

func (mqp *MysqlQueryPiece) GetSQL() (str *string) {
	return mqp.QuerySQL
}

func (pmqp *PooledMysqlQueryPiece) Recovery() {
	// pmqp.sliceBufferPool.Enqueue(pmqp.jsonContent[:0])
	pmqp.jsonContent = nil
	pmqp.recoverPool.Enqueue(pmqp)
}
