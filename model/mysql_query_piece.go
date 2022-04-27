package model

import (
	"github.com/pingcap/tidb/util/hack"
)

// MysqlQueryPiece 查询信息
type MysqlQueryPiece struct {
	BaseQueryPiece

	SessionID  *string `json:"-"`
	ClientHost *string `json:"cip"`
	ClientPort int     `json:"cport"`

	VisitUser    *string `json:"user"`
	VisitDB      *string `json:"db"`
	QuerySQL     *string `json:"sql"`
	CostTimeInMS int64   `json:"cms"`
	// SQL执行返回状态 1代表成功，-1代表失败, 0代表未知
	ResponseStatus int `json:"qrs"`
	// SQL执行返回信息，成功的时候代表影响行数，失败的时候代表错误码
	ResponseInfo int `json:"qri"`
}

func (mqp *MysqlQueryPiece) String() *string {
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
