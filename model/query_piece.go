package model

import (
	"bytes"
	"encoding/json"
	jsoniter "github.com/json-iterator/go"
	"github.com/pingcap/tidb/util/hack"
	"time"
)

var jsonIterator = jsoniter.ConfigCompatibleWithStandardLibrary

type QueryPiece interface {
	String() *string
	Bytes() []byte
	GetSQL() *string
	NeedSyncSend() bool
	Recovery()
}

// BaseQueryPiece 查询信息
type BaseQueryPiece struct {
	SyncSend          bool    `json:"-"`
	ServerIP          *string `json:"sip"`
	ServerPort        int     `json:"sport"`
	CapturePacketRate float64 `json:"cpr"`
	EventTime         int64   `json:"bt"`
	jsonContent       []byte  `json:"-"`
}

const (
	millSecondUnit = int64(time.Millisecond)
)

var (
	mqpp = NewMysqlQueryPiecePool()
)

var commonBaseQueryPiece = &BaseQueryPiece{}

func NewBaseQueryPiece(
	serverIP *string, serverPort int, capturePacketRate float64) (
	bqp *BaseQueryPiece) {
	bqp = commonBaseQueryPiece
	bqp.ServerIP = serverIP
	bqp.ServerPort = serverPort
	bqp.SyncSend = false
	bqp.CapturePacketRate = capturePacketRate
	bqp.EventTime = time.Now().UnixNano() / millSecondUnit

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

	bqp.jsonContent = marsharQueryPieceMonopolize(bqp)
	return bqp.jsonContent
}

func (bqp *BaseQueryPiece) GetSQL() (*string) {
	return nil
}

func (bqp *BaseQueryPiece) Recovery() {
}

func marsharQueryPieceShareMemory(qp interface{}, cacheBuffer []byte) []byte {

	buffer := bytes.NewBuffer(cacheBuffer)
	err := json.NewEncoder(buffer).Encode(qp)
	if err != nil {
		return []byte(err.Error())
	}

	return buffer.Bytes()
}

func marsharQueryPieceMonopolize(qp interface{}) (content []byte) {
	content, err := jsonIterator.Marshal(qp)
	if err != nil {
		return []byte(err.Error())
	}

	return content
}
