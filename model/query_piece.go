package model

import (
	"encoding/json"
)

type QueryPiece interface {
	String() string
	Bytes() []byte
	GetSQL() string
}

// MysqlQueryPiece 查询信息
type MysqlQueryPiece struct {
	SessionID    string   `json:"sid"`
	ClientHost   string  `json:"-"`
	ServerIP     string  `json:"sip"`
	ServerPort   int     `json:"sport"`
	VisitUser    *string `json:"user"`
	VisitDB      *string `json:"db"`
	QuerySQL     *string `json:"sql"`
	BeginTime    string  `json:"bt"`
	CostTimeInMS int64  `json:"cms"`
}

func (qp *MysqlQueryPiece) String() (str string) {
	content, err := json.Marshal(qp)
	if err != nil {
		return err.Error()
	}

	return string(content)
}

func (qp *MysqlQueryPiece) Bytes() (bytes []byte) {
	content, err := json.Marshal(qp)
	if err != nil {
		return []byte(err.Error())
	}

	return content
}

func (qp *MysqlQueryPiece) GetSQL() (str string) {
	if qp.QuerySQL != nil {
		return *qp.QuerySQL
	}
	return ""
}
