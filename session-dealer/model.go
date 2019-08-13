package session_dealer

import "github.com/zr-hebo/sniffer-agent/model"

type ConnSession interface {
	ReadFromClient(bytes []byte)
	ReadFromServer(bytes []byte)
	ResetBeginTime()
	GenerateQueryPiece() (qp model.QueryPiece)
}
