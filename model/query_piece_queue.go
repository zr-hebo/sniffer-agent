package model

import (
	"sync"
)

type mysqlQueryPiecePool struct {
	queue  []*PooledMysqlQueryPiece
	lock sync.Mutex
}

func NewMysqlQueryPiecePool() (mqpp *mysqlQueryPiecePool) {
	return &mysqlQueryPiecePool{
		queue: make([]*PooledMysqlQueryPiece, 0, 5000),
	}
}

func (mqpp *mysqlQueryPiecePool) Enqueue(pmqp *PooledMysqlQueryPiece)  {
	mqpp.lock.Lock()
	defer mqpp.lock.Unlock()

	mqpp.queue = append(mqpp.queue, pmqp)
}

func (mqpp *mysqlQueryPiecePool) Dequeue() (pmqp *PooledMysqlQueryPiece)  {
	mqpp.lock.Lock()
	defer mqpp.lock.Unlock()

	if len(mqpp.queue) < 1 {
		return nil
	}

	pmqp = mqpp.queue[0]
	mqpp.queue = mqpp.queue[1:]
	return
}