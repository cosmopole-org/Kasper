package trx

import (
	"crypto/rsa"
	"kasper/src/abstract/models/update"
)

type IModel[T any] interface {
	Type() string
	Parse(ITrx) T
}

type ITrx interface {
	DelKey(key string)
	HasObj(typ string, key string) bool
	GetIndex(typ string, objId string, fromColumn string, toColumn string) string
	HasIndex(typ string, objId string, fromColumn string, toColumn string) bool
	GetColumn(typ string, objId string, columnName string) []byte
	GetLink(key string) string
	PutLink(key string, value string)
	PutBytes(key string, value []byte)
	GetBytes(key string) []byte
	PutString(key string, value string)
	GetString(key string) string
	GetObj(typ string, key string) map[string][]byte
	PutObj(typ string, key string, keys map[string][]byte)
	PutJson(key string, path string, jsonObj any)
	DelJson(key string)
	GetJson(key string, path string) any
	GetPriKey(string) *rsa.PrivateKey
	GetPubKey(string) *rsa.PublicKey
	Updates() []update.Update
	Commit()
}
