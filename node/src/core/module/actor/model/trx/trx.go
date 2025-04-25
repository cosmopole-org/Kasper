package module_trx

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"log"

	"github.com/dgraph-io/badger"
)

type TrxWrapper struct {
	core    core.ICore
	dbTrx   *badger.Txn
	Changes []update.Update
}

func NewTrx(core core.ICore, storage storage.IStorage, readonly bool) trx.ITrx {
	tw := &TrxWrapper{core: core, Changes: []update.Update{}}
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

func (tw *TrxWrapper) indexJson(key string, path string, obj map[string]any, merge bool) {
	old := map[string]any{}
	if merge {
		var e error
		old, e = tw.GetJson(key, path)
		if e != nil {
			old = map[string]any{}
			log.Println(e)
		}
	}
	for k, v := range obj {
		old[k] = v
	}
	b, _ := json.Marshal(obj)
	tw.PutBytes("json::"+key+"::"+path, b)
	for k, v := range obj {
		if v != nil {
			if m, ok := v.(map[string]any); ok {
				tw.indexJson(key, path+"."+k, m, merge)
			} else {
				b, _ := json.Marshal(obj)
				tw.PutBytes("json::"+key+"::"+path+"."+k, b)
			}
		}
	}
}

func (tw *TrxWrapper) PutJson(key string, path string, jsonObj any, merge bool) error {
	b, e := json.Marshal(jsonObj)
	if e != nil {
		return e
	}
	m := map[string]any{}
	e = json.Unmarshal(b, &m)
	if e != nil {
		return e
	}
	tw.indexJson(key, path, m, merge)
	return nil
}

func (tw *TrxWrapper) DelJson(key string, path string) {
	tw.DelKey("json::" + key + "::" + path)
}

func (tw *TrxWrapper) GetJson(key string, path string) (map[string]any, error) {
	b := tw.GetBytes("json::" + key + "::" + path)
	m := map[string]any{}
	e := json.Unmarshal(b, &m)
	if e != nil {
		return m, e
	}
	return m, nil
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

func (tw *TrxWrapper) GetLinksList(p string, offset int, count int) ([]string, error) {
	prefix := []byte("link::" + p)
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true
	opts.Prefix = prefix
	it := tw.dbTrx.NewIterator(opts)
	defer it.Close()
	m := []string{}
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		itemKey := item.Key()
		m = append(m, string(itemKey))
	}
	return m, nil
}

func (tw *TrxWrapper) GetObjList(typ string, objIds []string) (map[string]map[string][]byte, error) {
	m := map[string]map[string][]byte{}
	for _, id := range objIds {
		m[id] = tw.GetObj(typ, id)
	}
	return m, nil
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

func (tw *TrxWrapper) Updates() []update.Update {
	return tw.Changes
}
