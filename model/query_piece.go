package model

import (
	// "github.com/json-iterator/go"
	"bytes"
	"encoding/json"
	"github.com/pingcap/tidb/util/hack"
	"time"
)

type QueryPiece interface {
	String() *string
	Bytes() []byte
	GetSQL() *string
	NeedSyncSend() bool
	Recovery()
}

// BaseQueryPiece 查询信息
type BaseQueryPiece struct {
	SyncSend        bool    `json:"-"`
	ServerIP        *string `json:"sip"`
	ServerPort      int     `json:"sport"`
	ThrowPacketRate float64 `json:"tpr"`
	BeginTime       int64   `json:"bt"`
	jsonContent     []byte  `json:"-"`
}

const (
	millSecondUnit = int64(time.Millisecond)
)

var (
	mqpp                 = NewMysqlQueryPiecePool()
	localSliceBufferPool = NewSliceBufferPool("json cache", 2*1024*1024)
)

var commonBaseQueryPiece = &BaseQueryPiece{}

func NewBaseQueryPiece(
	serverIP *string, serverPort int, throwPacketRate float64) (
	bqp *BaseQueryPiece) {
	bqp = commonBaseQueryPiece
	bqp.ServerIP = serverIP
	bqp.ServerPort = serverPort
	bqp.SyncSend = false
	bqp.ThrowPacketRate = throwPacketRate
	bqp.BeginTime = time.Now().UnixNano() / millSecondUnit

	return
}

func (bqp *BaseQueryPiece) NeedSyncSend() (bool) {
	return bqp.SyncSend
}

func (bqp *BaseQueryPiece) SetNeedSyncSend(syncSend bool) {
	bqp.SyncSend = syncSend
}

func (bqp *BaseQueryPiece) String() (*string) {
	content := bqp.Bytes()
	contentStr := hack.String(content)
	return &contentStr
}

func (bqp *BaseQueryPiece) Bytes() (content []byte) {
	// content, err := json.Marshal(bqp)
	if bqp.jsonContent != nil && len(bqp.jsonContent) > 0 {
		return bqp.jsonContent
	}

	bqp.jsonContent = marsharQueryPiece(bqp)
	return bqp.jsonContent
}

func (bqp *BaseQueryPiece) GetSQL() (*string) {
	return nil
}

func (bqp *BaseQueryPiece) Recovery() {
}

func marsharQueryPiece(qp interface{}) []byte {
	var cacheBuffer = localSliceBufferPool.Dequeue()
	if len(cacheBuffer) > 0 {
		panic("there already have bytes in buffer")
	}

	buffer := bytes.NewBuffer(cacheBuffer)
	err := json.NewEncoder(buffer).Encode(qp)
	if err != nil {
		return []byte(err.Error())
	}

	return buffer.Bytes()
}
