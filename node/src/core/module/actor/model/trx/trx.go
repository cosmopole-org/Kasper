package module_trx

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"log"
	"sort"
	"strings"

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
	e := tw.dbTrx.Commit()
	if e != nil {
		log.Println("Error on committing:", e)
		log.Println("retrying commit in safe context")
		err := tw.core.Tools().Storage().KvDb().Update(func(txn *badger.Txn) error {
			for _, change := range tw.Changes {
				if change.Typ == "put" {
					if err := txn.Set([]byte(change.Key), change.Val); err != nil {
						return err
					}
				} else if change.Typ == "del" {
					if err := txn.Delete([]byte(change.Key)); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Println("Error on safe context committing:", err)
		}
	}
}

func (tw *TrxWrapper) DelKey(key string) {
	tw.dbTrx.Delete([]byte(key))
	tw.Changes = append(tw.Changes, update.Update{Typ: "del", Key: key})
}

func (tw *TrxWrapper) HasObj(typ string, key string) bool {
	_, e := tw.dbTrx.Get([]byte("obj::" + typ + "::" + key + "::|"))
	return e == nil
}

func (tw *TrxWrapper) GetIndex(typ string, fromColumn string, toColumn string, fromColumnVal string) string {
	item, e := tw.dbTrx.Get([]byte("index::" + typ + "::" + fromColumn + "::" + toColumn + "::" + fromColumnVal))
	if e != nil {
		return ""
	} else {
		var value []byte
		item.Value(func(val []byte) error {
			value = val
			return nil
		})
		return string(value)
	}
}

func (tw *TrxWrapper) PutIndex(typ string, fromColumn string, toColumn string, fromColumnVal string, toColumnVal []byte) {
	tw.dbTrx.Set([]byte("index::"+typ+"::"+fromColumn+"::"+toColumn+"::"+fromColumnVal), toColumnVal)
	tw.Changes = append(tw.Changes, update.Update{Typ: "put", Key: "index::" + typ + "::" + fromColumn + "::" + toColumn + "::" + fromColumnVal, Val: toColumnVal})
}

func (tw *TrxWrapper) HasIndex(typ string, fromColumn string, toColumn string, fromColumnVal string) bool {
	_, e := tw.dbTrx.Get([]byte("index::" + typ + "::" + fromColumn + "::" + toColumn + "::" + fromColumnVal))
	return e == nil
}

func (tw *TrxWrapper) GetLink(key string) string {
	item, e := tw.dbTrx.Get([]byte("link::" + key))
	if e != nil {
		return ""
	} else {
		var value []byte
		item.Value(func(val []byte) error {
			value = val
			return nil
		})
		return string(value)
	}
}

func (tw *TrxWrapper) PutLink(key string, value string) {
	tw.dbTrx.Set([]byte("link::"+key), []byte(value))
	tw.Changes = append(tw.Changes, update.Update{Typ: "put", Key: "link::" + key, Val: []byte(value)})
}

func (tw *TrxWrapper) PutBytes(key string, value []byte) {
	tw.dbTrx.Set([]byte(key), value)
	tw.Changes = append(tw.Changes, update.Update{Typ: "put", Key: key, Val: value})
}

func (tw *TrxWrapper) GetBytes(key string) []byte {
	item, e := tw.dbTrx.Get([]byte(key))
	if e == nil {
		var value []byte
		item.Value(func(val []byte) error {
			value = val
			return nil
		})
		return value
	}
	return []byte{}
}

func (tw *TrxWrapper) PutString(key string, value string) {
	tw.dbTrx.Set([]byte(key), []byte(value))
	tw.Changes = append(tw.Changes, update.Update{Typ: "put", Key: key, Val: []byte(value)})
}

func (tw *TrxWrapper) GetString(key string) string {
	item, e := tw.dbTrx.Get([]byte(key))
	if e != nil {
		return ""
	} else {
		var value []byte
		item.Value(func(val []byte) error {
			value = val
			return nil
		})
		return string(value)
	}
}

func (tw *TrxWrapper) GetByPrefix(p string) []string {
	prefix := []byte(p)
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
	return m
}

func (tw *TrxWrapper) GetObj(typ string, key string) map[string][]byte {
	prefix := []byte("obj::" + typ + "::" + key + "::")
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
		m[string(itemKey[len(prefix):])] = itemVal
		if err != nil {
			return map[string][]byte{}
		}
	}
	return m
}

func (tw *TrxWrapper) PutObj(typ string, key string, keys map[string][]byte) {
	prefix := "obj::" + typ + "::" + key + "::"
	keys["|"] = []byte{0x01}
	for k, v := range keys {
		tw.dbTrx.Set([]byte(prefix+k), v)
		tw.Changes = append(tw.Changes, update.Update{Typ: "put", Key: prefix + k, Val: v})
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
	keys := make([]string, 0, len(obj))
	for k, v := range obj {
		keys = append(keys, k)
		old[k] = v
	}
	sort.Strings(keys)
	b, _ := json.Marshal(obj)
	tw.PutBytes("json::"+key+"::"+path, b)
	for _, k := range keys {
		v := obj[k]
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
	if len(b) == 0 {
		return m, errors.New("json path not found")
	}
	e := json.Unmarshal(b, &m)
	if e != nil {
		return m, e
	}
	return m, nil
}

func (tw *TrxWrapper) GetObjList(typ string, objIds []string, queryMap map[string]string, meta ...int64) (map[string]map[string][]byte, error) {
	if (len(objIds) == 1) && (objIds[0] == "*") {
		objs := map[string]map[string][]byte{}
		prefix := []byte("obj::" + typ + "::")
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.Prefix = prefix
		it := tw.dbTrx.NewIterator(opts)
		defer it.Close()
		if len(meta) == 0 {
			temp := map[string][]byte{}
			tempId := ""
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				itemKey := item.Key()
				id := strings.Split(string(itemKey[len(prefix):]), "::")[0]
				var itemVal []byte
				err := item.Value(func(v []byte) error {
					itemVal = v
					return nil
				})
				if tempId != id {
					matched := true
					if len(queryMap) > 0 {
						for k, v := range queryMap {
							if v != string(temp[k]) {
								matched = false
							}
						}
					}
					if _, ok := temp["|"]; ok && matched && (tempId != "") {
						objs[tempId] = temp
					}
					temp = map[string][]byte{}
					tempId = id
				}
				temp[string(itemKey)[len(string(prefix))+len(id)+len("::"):]] = itemVal
				if err != nil {
					return nil, err
				}
			}
			matched := true
			if len(queryMap) > 0 {
				for k, v := range queryMap {
					if (len(temp[k]) == 0) || (v != string(temp[k])) {
						matched = false
					}
				}
			}
			if _, ok := temp["|"]; ok && matched && (tempId != "") {
				objs[tempId] = temp
			}
		} else if len(meta) == 1 {
			index := int64(0)
			offset := meta[0]
			temp := map[string][]byte{}
			tempId := ""
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				itemKey := item.Key()
				id := strings.Split(string(itemKey[len(prefix):]), "::")[0]
				var itemVal []byte
				err := item.Value(func(v []byte) error {
					itemVal = v
					return nil
				})
				if tempId != id {
					matched := true
					if len(queryMap) > 0 {
						for k, v := range queryMap {
							if v != string(temp[k]) {
								matched = false
							}
						}
					}
					if _, ok := temp["|"]; ok && matched && (tempId != "") {
						if index < offset {
							index++
							temp = map[string][]byte{}
							tempId = id
							continue
						}
						index++
						objs[tempId] = temp
					}
					temp = map[string][]byte{}
					tempId = id
				}
				temp[string(itemKey)[len(string(prefix))+len(id)+len("::"):]] = itemVal
				if err != nil {
					return nil, err
				}
			}
			matched := true
			if len(queryMap) > 0 {
				for k, v := range queryMap {
					if (len(temp[k]) == 0) || (v != string(temp[k])) {
						matched = false
					}
				}
			}
			if _, ok := temp["|"]; ok && matched && (tempId != "") {
				if index >= offset {
					objs[tempId] = temp
				}
			}
		} else if len(meta) == 2 {
			index := int64(0)
			offset := meta[0]
			count := meta[1]
			temp := map[string][]byte{}
			tempId := ""
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				itemKey := item.Key()
				id := strings.Split(string(itemKey[len(prefix):]), "::")[0]
				var itemVal []byte
				err := item.Value(func(v []byte) error {
					itemVal = v
					return nil
				})
				if tempId != id {
					matched := true
					if len(queryMap) > 0 {
						for k, v := range queryMap {
							if v != string(temp[k]) {
								matched = false
							}
						}
					}
					if _, ok := temp["|"]; ok && matched && (tempId != "") {
						log.Println("id", id, index, offset, count)
						if index < offset {
							index++
							temp = map[string][]byte{}
							tempId = id
							continue
						}
						if index >= (offset + count) {
							break
						}
						index++
						log.Println("id second", id, index, offset, count)
						objs[tempId] = temp
					}
					temp = map[string][]byte{}
					tempId = id
				}
				temp[string(itemKey)[len(string(prefix))+len(id)+len("::"):]] = itemVal
				if err != nil {
					return nil, err
				}
			}
			matched := true
			if len(queryMap) > 0 {
				for k, v := range queryMap {
					if (len(temp[k]) == 0) || (v != string(temp[k])) {
						matched = false
					}
				}
			}
			if _, ok := temp["|"]; ok && matched && (tempId != "") {
				if index >= offset && index < (offset+count) {
					objs[tempId] = temp
				}
			}
		}
		return objs, nil
	} else {
		m := map[string]map[string][]byte{}
		for _, id := range objIds {
			m[id] = tw.GetObj(typ, id)
		}
		return m, nil
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
		m = append(m, string(itemKey)[len("link::"):])
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

func (tw *TrxWrapper) GetPubKey(key string) *rsa.PublicKey {
	res := tw.GetString("obj::User::" + key + "::publicKey")
	if res == "" {
		return nil
	} else {
		pemData := "-----BEGIN RSA PUBLIC KEY-----\n" + res + "\n-----END RSA PUBLIC KEY-----\n"
		block, _ := pem.Decode([]byte(pemData))
		if block == nil || block.Type != "RSA PUBLIC KEY" {
			log.Println("failed to decode PEM block containing public key")
			return nil
		}
		publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			log.Println(err)
			return nil
		}
		return publicKey
	}
}

func (tw *TrxWrapper) Updates() []update.Update {
	return tw.Changes
}
