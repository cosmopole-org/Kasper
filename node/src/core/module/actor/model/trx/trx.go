package module_trx

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models"
	"kasper/src/abstract/models/core"
	"log"

	"github.com/dgraph-io/badger"
)

type TrxWrapper struct {
	core    core.ICore
	dbTrx   *badger.Txn
	Changes []models.Update
}

func NewTrx(core core.ICore, storage storage.IStorage, readonly bool) models.ITrx {
	tw := &TrxWrapper{core: core, Changes: []models.Update{}}
	tw.dbTrx = storage.KvDb().NewTransaction(!readonly)
	return tw
}

func (tw *TrxWrapper) GetColumn(typ string, objId string, columnName string) []byte {
	item, e := tw.dbTrx.Get([]byte("obj::" + typ + "::" + objId + "::" + columnName))
	if e != nil {
		return []byte{}
	} else {
		res := []byte{}
		item.Value(func(val []byte) error {
			res = val
			return nil
		})
		return res
	}
}

func (tw *TrxWrapper) Commit() {
	tw.dbTrx.Commit()
}

func (tw *TrxWrapper) DelKey(key string) {
	tw.dbTrx.Delete([]byte(key))
}

func (tw *TrxWrapper) HasObj(typ string, key string) bool {
	_, e := tw.dbTrx.Get([]byte("obj::" + typ + "::" + key))
	return e == nil
}

func (tw *TrxWrapper) GetIndex(typ string, objId string, fromColumn string, toColumn string) string {
	item, e := tw.dbTrx.Get([]byte("index::" + typ + "::" + objId + "::" + fromColumn + "::" + toColumn))
	if e != nil {
		return ""
	} else {
		return item.String()
	}
}

func (tw *TrxWrapper) HasIndex(typ string, objId string, fromColumn string, toColumn string) bool {
	_, e := tw.dbTrx.Get([]byte("index::" + typ + "::" + objId + "::" + fromColumn + "::" + toColumn))
	return e == nil
}

func (tw *TrxWrapper) GetLink(key string) string {
	item, e := tw.dbTrx.Get([]byte("link::" + key))
	if e != nil {
		return ""
	} else {
		return item.String()
	}
}

func (tw *TrxWrapper) PutLink(key string, value string) {
	tw.dbTrx.Set([]byte("link::"+key), []byte(value))
}

func (tw *TrxWrapper) PutBytes(key string, value []byte) {
	tw.dbTrx.Set([]byte(key), value)
}

func (tw *TrxWrapper) GetBytes(key string) []byte {
	item, e := tw.dbTrx.Get([]byte(key))
	val := make([]byte, item.ValueSize())
	if e == nil {
		item.ValueCopy(val)
	}
	return val
}

func (tw *TrxWrapper) PutString(key string, value string) {
	tw.dbTrx.Set([]byte(key), []byte(value))
}

func (tw *TrxWrapper) GetString(key string) string {
	item, e := tw.dbTrx.Get([]byte(key))
	if e != nil {
		return ""
	} else {
		return item.String()
	}
}

func (tw *TrxWrapper) GetObj(typ string, key string) map[string][]byte {
	prefix := []byte("obj::" + typ + "::" + key)
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true
	opts.Prefix = prefix
	it := tw.dbTrx.NewIterator(opts)
	defer it.Close()
	m := map[string][]byte{}
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		itemKey := item.Key()
		var itemVal []byte
		err := item.Value(func(v []byte) error {
			itemVal = v
			return nil
		})
		m[string(itemKey)] = itemVal
		if err != nil {
			return map[string][]byte{}
		}
	}
	return m
}

func (tw *TrxWrapper) PutObj(typ string, key string, keys map[string][]byte) {
	prefix := "obj::" + typ + "::" + key + "::"
	for k, v := range keys {
		tw.dbTrx.Set([]byte(prefix+k), v)
	}
}

func (tw *TrxWrapper) PutJson(key string, path string, jsonObj any) {

}

func (tw *TrxWrapper) DelJson(key string) {

}

func (tw *TrxWrapper) GetJson(key string, path string) any {
	return struct{}{}
}

func (tw *TrxWrapper) GetPriKey(key string) *rsa.PrivateKey {
	res := tw.GetString("obj::User::" + key + "::privateKey")
	if res == "" {
		return nil
	} else {
		pemData := "-----BEGIN RSA PRIVATE KEY-----\n" + res + "\n-----END RSA PRIVATE KEY-----\n"
		block, _ := pem.Decode([]byte(pemData))
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			log.Println("failed to decode PEM block containing private key")
			return nil
		}
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			log.Println(err)
			return nil
		}
		return privateKey
	}
}

func (tw *TrxWrapper) GetPubKey(key string) *rsa.PublicKey {
	res := tw.GetString("obj::User::" + key + "::publicKey")
	if res == "" {
		return nil
	} else {
		pemData := "-----BEGIN RSA PUBLIC KEY-----\n" + res + "\n-----END RSA PUBLIC KEY-----\n"
		block, _ := pem.Decode([]byte(pemData))
		if block == nil || block.Type != "PUBLIC KEY" {
			log.Println("failed to decode PEM block containing public key")
			return nil
		}
		pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			log.Println(err)
			return nil
		}
		publicKey, ok := pubKeyInterface.(*rsa.PublicKey)
		if !ok {
			log.Println("not RSA public key")
			return nil
		}
		return publicKey
	}
}

func (tw *TrxWrapper) Updates() []models.Update {
	return tw.Changes
}
