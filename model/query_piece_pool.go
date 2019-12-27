package model

import (
	"sync"
	"time"
)

type PooledMysqlQueryPiece struct {
	MysqlQueryPiece
	recoverPool     *mysqlQueryPiecePool
}

func NewPooledMysqlQueryPiece(
	sessionID, clientIP, visitUser, visitDB, serverIP *string,
	clientPort, serverPort int, throwPacketRate float64, stmtBeginTimeNano int64) (
	pmqp *PooledMysqlQueryPiece) {
	pmqp = mqpp.Dequeue()

	pmqp.SessionID = sessionID
	pmqp.ClientHost = clientIP
	pmqp.ClientPort = clientPort
	pmqp.ServerIP = serverIP
	pmqp.ServerPort = serverPort
	pmqp.VisitUser = visitUser
	pmqp.VisitDB = visitDB
	pmqp.SyncSend = false
	pmqp.CapturePacketRate = throwPacketRate
	pmqp.EventTime = stmtBeginTimeNano / millSecondUnit
	pmqp.CostTimeInMS = (time.Now().UnixNano() - stmtBeginTimeNano) / millSecondUnit
	pmqp.recoverPool = mqpp

	return
}

func (pmqp *PooledMysqlQueryPiece) Recovery() {
	pmqp.jsonContent = nil
	pmqp.recoverPool.Enqueue(pmqp)
}

func (pmqp *PooledMysqlQueryPiece) Bytes() (content []byte) {
	// content, err := json.Marshal(mqp)
	if len(pmqp.jsonContent) > 0 {
		return pmqp.jsonContent
	}

	pmqp.GenerateJsonBytes()
	return pmqp.jsonContent
}

func (pmqp *PooledMysqlQueryPiece) GenerateJsonBytes() {
	pmqp.jsonContent = marsharQueryPieceMonopolize(pmqp)
	return
}

type mysqlQueryPiecePool struct {
	queue  chan *PooledMysqlQueryPiece
	lock sync.Mutex
}

func NewMysqlQueryPiecePool() (mqpp *mysqlQueryPiecePool) {
	return &mysqlQueryPiecePool{
		queue: make(chan *PooledMysqlQueryPiece, 512),
	}
}

func (mqpp *mysqlQueryPiecePool) Enqueue(pmqp *PooledMysqlQueryPiece)  {
	mqpp.lock.Lock()
	defer mqpp.lock.Unlock()

	select {
	case mqpp.queue <- pmqp:
		return
	default:
		pmqp = nil
		return
	}
}

func (mqpp *mysqlQueryPiecePool) Dequeue() (pmqp *PooledMysqlQueryPiece)  {
	mqpp.lock.Lock()
	defer mqpp.lock.Unlock()

	select {
	case pmqp = <- mqpp.queue:
		return

	default:
		pmqp = &PooledMysqlQueryPiece{
			MysqlQueryPiece: MysqlQueryPiece{},
		}
		return
	}
}