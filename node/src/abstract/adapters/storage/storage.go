package storage

import (
	"kasper/src/abstract/models/packet"
	"kasper/src/abstract/models/trx"

	"github.com/dgraph-io/badger"
)

type IStorage interface {
	StorageRoot() string
	KvDb() *badger.DB
	GenId(t trx.ITrx, origin string) string
	LogTimeSieries(pointId string, userId string, data string, timeVal int64)
	ReadPointLogs(pointId string, beforeTime string, count int) []packet.LogPacket
}
