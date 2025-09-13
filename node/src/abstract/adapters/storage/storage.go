package storage

import (
	"kasper/src/abstract/models/packet"
	"kasper/src/abstract/models/trx"

	"database/sql"
	"github.com/blevesearch/bleve/v2"
	"github.com/dgraph-io/badger"
)

type IStorage interface {
	StorageRoot() string
	KvDb() *badger.DB
	TsDb() *sql.DB
	Searcher() bleve.Index
	GenId(t trx.ITrx, origin string) string
	LogTimeSieries(pointId string, userId string, data string, timeVal int64) packet.LogPacket
	UpdateLog(pointId string, userId string, signalId string, data string, timeVal int64) packet.LogPacket
	ReadPointLogs(pointId string, beforeTime int64, count int) []packet.LogPacket
	SearchPointLogs(pointId string, quest string) []packet.LogPacket
}
