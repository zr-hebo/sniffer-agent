package session_dealer

import "github.com/zr-hebo/sniffer-agent/model"

type ConnSession interface {
	ReadFromClient(bytes []byte)
	ReadFromServer(bytes []byte)
	GenerateQueryPiece() (qp model.QueryPiece)
}
