package session_dealer

import "github.com/zr-hebo/sniffer-agent/model"

type ConnSession interface {
	ReadFromClient(seqID int64, bytes []byte)
	ReadFromServer(bytes []byte)
	ResetBeginTime()
	GenerateQueryPiece() (qp model.QueryPiece)
}
