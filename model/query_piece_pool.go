package model

import (
	"github.com/zr-hebo/sniffer-agent/util"
	"sync"
	"time"
)

var (
	localSliceBufferPool = util.NewSliceBufferPool("json cache", (128+1)*1024)
)

type PooledMysqlQueryPiece struct {
	MysqlQueryPiece
	recoverPool     *mysqlQueryPiecePool
	sliceBufferPool *util.SliceBufferPool
}

func NewPooledMysqlQueryPiece(
	sessionID, clientIP, visitUser, visitDB, serverIP *string,
	clientPort, serverPort int, throwPacketRate float64, stmtBeginTime int64) (
	pmqp *PooledMysqlQueryPiece) {
	pmqp = mqpp.Dequeue()

	pmqp.sliceBufferPool = localSliceBufferPool
	nowInMS := time.Now().UnixNano() / millSecondUnit
	pmqp.SessionID = sessionID
	pmqp.ClientHost = clientIP
	pmqp.ClientPort = clientPort
	pmqp.ServerIP = serverIP
	pmqp.ServerPort = serverPort
	pmqp.VisitUser = visitUser
	pmqp.VisitDB = visitDB
	pmqp.SyncSend = false
	pmqp.CapturePacketRate = throwPacketRate
	pmqp.EventTime = stmtBeginTime
	pmqp.CostTimeInMS = nowInMS - stmtBeginTime
	pmqp.recoverPool = mqpp

	return
}

func (pmqp *PooledMysqlQueryPiece) Recovery() {
	if pmqp.sliceBufferPool != nil {
		pmqp.sliceBufferPool.Enqueue(pmqp.jsonContent[:0])
	}
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
	if pmqp.sliceBufferPool == nil {
		pmqp.jsonContent = marsharQueryPieceMonopolize(pmqp)
		return
	}

	var cacheBuffer = pmqp.sliceBufferPool.Dequeue()
	if len(cacheBuffer) > 0 {
		panic("there already have bytes in buffer")
	}

	pmqp.jsonContent = marsharQueryPieceShareMemory(pmqp, cacheBuffer)
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