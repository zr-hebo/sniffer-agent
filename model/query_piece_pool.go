package model

import (
	"sync"
)

type mysqlQueryPiecePool struct {
	queue  chan *PooledMysqlQueryPiece
	lock sync.Mutex
}

func NewMysqlQueryPiecePool() (mqpp *mysqlQueryPiecePool) {
	return &mysqlQueryPiecePool{
		queue: make(chan *PooledMysqlQueryPiece, 1024),
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