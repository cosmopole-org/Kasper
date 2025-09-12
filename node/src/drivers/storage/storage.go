package tool_storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/packet"
	"kasper/src/abstract/models/trx"
	"log"
	"os"
	"sync"

	"github.com/dgraph-io/badger"

	bleve "github.com/blevesearch/bleve/v2"
	gocql "github.com/gocql/gocql"
)

type StorageManager struct {
	core        core.ICore
	storageRoot string
	kvdb        *badger.DB
	tsdb        *gocql.Session
	searcher    bleve.Index
	lock        sync.Mutex
}

func (sm *StorageManager) StorageRoot() string {
	return sm.storageRoot
}
func (sm *StorageManager) KvDb() *badger.DB {
	return sm.kvdb
}
func (sm *StorageManager) Searcher() bleve.Index {
	return sm.searcher
}

func (sm *StorageManager) LogTimeSieries(pointId string, userId string, data string, timeVal int64) packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	id := gocql.TimeUUID()
	if err := sm.tsdb.Query(`INSERT INTO storage(id, point_id, user_id, data, time, edited) VALUES (?, ?, ?, ?, ?, ?)`,
		id, pointId, userId, data, timeVal, false).WithContext(ctx).Exec(); err != nil {
		log.Fatal(err)
	}
	packet := packet.LogPacket{Id: id.String(), UserId: userId, Data: data, PointId: pointId, Time: timeVal, Edited: false}
	sm.searcher.Index(packet.Id, packet)
	return packet
}

func (sm *StorageManager) UpdateLog(pointId string, userId string, signalId string, data string, timeVal int64) packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	id := gocql.TimeUUID()
	if err := sm.tsdb.Query(`UPDATE storage SET data = ?, time = ?, edited = ? WHERE user_id = ? IF id = ?;`, data, timeVal, true, userId, id).WithContext(ctx).Exec(); err != nil {
		log.Println(err)
	}
	packet := packet.LogPacket{Id: id.String(), UserId: userId, Data: data, PointId: pointId, Time: timeVal, Edited: true}
	sm.searcher.Index(packet.Id, packet)
	return packet
}

func (sm *StorageManager) ReadPointLogs(pointId string, beforeTime string, count int) []packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	var scanner gocql.Scanner
	if beforeTime == "" {
		scanner = sm.tsdb.Query(`SELECT id, user_id, data, time, edited FROM storage WHERE point_id = ? limit ? ALLOW FILTERING`, pointId, count).WithContext(ctx).Iter().Scanner()
	} else {
		uuid, err := gocql.ParseUUID(beforeTime)
		if err != nil {
			fmt.Println("Invalid UUID:", err)
			return []packet.LogPacket{}
		}
		scanner = sm.tsdb.Query(`SELECT id, user_id, data, time, edited FROM storage WHERE point_id = ? and id < ? limit ? ALLOW FILTERING`, pointId, uuid, count).WithContext(ctx).Iter().Scanner()
	}
	logs := []packet.LogPacket{}
	for scanner.Next() {
		var id gocql.UUID
		var userId string
		var data string
		var timeVal int64
		var edited bool
		err := scanner.Scan(&id, &userId, &data, &timeVal, &edited)
		if err != nil {
			log.Fatal(err)
		}
		logs = append(logs, packet.LogPacket{Id: id.String(), UserId: userId, Data: data, PointId: pointId, Time: timeVal, Edited: edited})
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return logs
}

func (sm *StorageManager) SearchPointLogs(pointId string, quest string) []packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()

	query := bleve.NewQueryStringQuery(quest)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Explain = true
	searchRequest.Fields = []string{"data"}
	searchResult, err := sm.searcher.Search(searchRequest)
	if err != nil {
		log.Println(err)
		return []packet.LogPacket{}
	}
	ids := []string{}
	for _, hit := range searchResult.Hits {
		ids = append(ids, hit.ID)
	}
	var scanner gocql.Scanner
	scanner = sm.tsdb.Query(`SELECT id, user_id, data, time, edited FROM storage WHERE point_id = ? and id in ? ALLOW FILTERING`, pointId, ids).WithContext(ctx).Iter().Scanner()
	logs := []packet.LogPacket{}
	for scanner.Next() {
		var id gocql.UUID
		var userId string
		var data string
		var timeVal int64
		var edited bool
		err := scanner.Scan(&id, &userId, &data, &timeVal, &edited)
		if err != nil {
			log.Fatal(err)
		}
		logs = append(logs, packet.LogPacket{Id: id.String(), UserId: userId, Data: data, PointId: pointId, Time: timeVal, Edited: edited})
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return logs
}

func (sm *StorageManager) GenId(t trx.ITrx, origin string) string {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if origin == "global" {
		item := t.GetBytes("globalIdCounter")
		var counter int64 = 0
		if len(item) == 0 {
			counter = 0
		} else {
			counter = int64(binary.BigEndian.Uint64(item))
		}
		counter++
		nextB := [8]byte{}
		binary.BigEndian.PutUint64(nextB[:], uint64(counter))
		t.PutBytes("globalIdCounter", nextB[:])
		return fmt.Sprintf("%d@%s", counter, origin)
	} else {
		trx := sm.kvdb.NewTransaction(true)
		defer trx.Commit()
		item, err := trx.Get([]byte("localIdCounter"))
		var counter int64 = 0
		if err != nil {
			counter = 0
		} else {
			var b []byte
			item.Value(func(val []byte) error {
				b = val
				return nil
			})
			counter = int64(binary.BigEndian.Uint64(b))
		}
		counter++
		nextB := [8]byte{}
		binary.BigEndian.PutUint64(nextB[:], uint64(counter))
		trx.Set([]byte("localIdCounter"), nextB[:])
		return fmt.Sprintf("%d@%s", counter, origin)
	}
}

func NewStorage(core core.ICore, storageRoot string, baseDbPath string, logsDbPath string, searcherDbPath string) *StorageManager {
	log.Println("connecting to database...")
	os.MkdirAll(baseDbPath, os.ModePerm)
	kvdb, err := badger.Open(badger.DefaultOptions(baseDbPath).WithSyncWrites(true))
	if err != nil {
		panic(err)
	}
	cluster := gocql.NewCluster(logsDbPath)
	cluster.Keyspace = "kasper"
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()
	if err != nil {
		panic(err)
	}
	var searcher bleve.Index
	if _, err := os.Stat(searcherDbPath); errors.Is(err, os.ErrNotExist) {
		mapping := bleve.NewIndexMapping()
		searcher, err = bleve.New(searcherDbPath, mapping)
		if err != nil {
			panic(err)
		}
	} else {
		searcher, err = bleve.Open(searcherDbPath)
		if err != nil {
			panic(err)
		}
	}
	return &StorageManager{core: core, tsdb: session, kvdb: kvdb, searcher: searcher, storageRoot: storageRoot}
}
