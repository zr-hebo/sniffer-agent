package model

import (
	"bytes"

	"encoding/json"
	// "github.com/json-iterator/go"
	"time"

	"github.com/pingcap/tidb/util/hack"
)

// var json = jsoniter.ConfigCompatibleWithStandardLibrary

type QueryPiece interface {
	String() *string
	Bytes() []byte
	GetSQL() *string
	NeedSyncSend() bool
	Recovery()
}

// MysqlQueryPiece 查询信息
type MysqlQueryPiece struct {
	SessionID    *string `json:"cid"`
	ClientHost   *string `json:"-"`
	ClientPort   int     `json:"-"`
	SyncSend     bool    `json:"-"`
	ServerIP     *string `json:"sip"`
	ServerPort   int     `json:"sport"`
	VisitUser    *string `json:"user"`
	VisitDB      *string `json:"db"`
	QuerySQL     *string `json:"sql"`
	ThrowPacketRate    float64  `json:"tpr"`
	BeginTime    int64  `json:"bt"`
	CostTimeInMS int64   `json:"cms"`

	jsonContent     []byte    `json:"-"`
}

type PooledMysqlQueryPiece struct {
	MysqlQueryPiece
	recoverPool *mysqlQueryPiecePool
	sliceBufferPool *sliceBufferPool
}

const (
	millSecondUnit = int64(time.Millisecond)
)

var (
	mqpp = NewMysqlQueryPiecePool()
	localSliceBufferPool = NewSliceBufferPool("json cache", 2*1024*1024)
)

func NewPooledMysqlQueryPiece(
	sessionID, clientIP, visitUser, visitDB, clientHost, serverIP *string,
	clientPort, serverPort int, throwPacketRate float64, stmtBeginTime int64) (
	mqp *PooledMysqlQueryPiece) {
	mqp = mqpp.Dequeue()

	nowInMS := time.Now().UnixNano() / millSecondUnit
	mqp.SessionID = sessionID
	mqp.ClientHost = clientIP
	mqp.ClientPort = clientPort
	mqp.ClientHost = clientHost
	mqp.ServerIP = serverIP
	mqp.ServerPort = serverPort
	mqp.VisitUser = visitUser
	mqp.VisitDB = visitDB
	mqp.SyncSend = false
	mqp.ThrowPacketRate = throwPacketRate
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

	var cacheBuffer = localSliceBufferPool.Dequeue()
	buffer := bytes.NewBuffer(cacheBuffer)
	err := json.NewEncoder(buffer).Encode(mqp)
	if err != nil {
		mqp.jsonContent = []byte(err.Error())

	} else {
		mqp.jsonContent = buffer.Bytes()
	}

	return mqp.jsonContent
}

func (mqp *MysqlQueryPiece) GetSQL() (str *string) {
	return mqp.QuerySQL
}

func (mqp *MysqlQueryPiece) NeedSyncSend() (bool) {
	return mqp.SyncSend
}

func (mqp *MysqlQueryPiece) SetNeedSyncSend(syncSend bool) {
	mqp.SyncSend = syncSend
}

func (pmqp *PooledMysqlQueryPiece) Recovery() {
	pmqp.recoverPool.Enqueue(pmqp)
	pmqp.sliceBufferPool.Enqueue(pmqp.jsonContent[:0])
	pmqp.jsonContent = nil
}
