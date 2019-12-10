package model

import (
	"github.com/pingcap/tidb/util/hack"
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
