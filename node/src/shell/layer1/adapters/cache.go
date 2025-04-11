package adapters

import (
	"kasper/src/abstract"

	"gorm.io/gorm"
)

type ICache interface {
	Keys(mainPart string) []string
	GenId(*gorm.DB, string) string
	Infra() any
	DoCacheTrx() ICacheTrx
	Put(key string, value string)
	Get(key string) string
	Del(key string)
}

type ICacheTrx interface {
	Put(key string, value string)
	Del(key string)
	Updates() []abstract.CacheUpdate
}

func NewUpdatePut(key string, val string) abstract.CacheUpdate {
	return abstract.CacheUpdate{Typ: "put", Key: key, Val: val}
}
func NewUpdateDel(key string) abstract.CacheUpdate {
	return abstract.CacheUpdate{Typ: "put", Key: key}
}
