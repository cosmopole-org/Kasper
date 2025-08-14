package tool_storage

import (
	"context"
	"encoding/binary"
	"fmt"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/packet"
	"kasper/src/abstract/models/trx"
	"log"
	"os"
	"sync"

	"github.com/dgraph-io/badger"

	gocql "github.com/gocql/gocql"
)

type StorageManager struct {
	core        core.ICore
	storageRoot string
	kvdb        *badger.DB
	tsdb        *gocql.Session
	lock        sync.Mutex
}

func (sm *StorageManager) StorageRoot() string {
	return sm.storageRoot
}
func (sm *StorageManager) KvDb() *badger.DB {
	return sm.kvdb
}

func (sm *StorageManager) LogTimeSieries(pointId string, userId string, data string, timeVal int64) packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	id := gocql.TimeUUID()
	if err := sm.tsdb.Query(`INSERT INTO storage(id, point_id, user_id, data, time) VALUES (?, ?, ?, ?, ?)`,
		id, pointId, userId, data, timeVal).WithContext(ctx).Exec(); err != nil {
		log.Fatal(err)
	}
	return packet.LogPacket{Id: id.String(), UserId: userId, Data: data, PointId: pointId, Time: timeVal}
}

func (sm *StorageManager) ReadPointLogs(pointId string, beforeTime string, count int) []packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	var scanner gocql.Scanner
	if beforeTime == "" {
		scanner = sm.tsdb.Query(`SELECT id, user_id, data, time FROM storage WHERE point_id = ? limit ? ALLOW FILTERING`, pointId, count).WithContext(ctx).Iter().Scanner()
	} else {
		uuid, err := gocql.ParseUUID(beforeTime)
		if err != nil {
			fmt.Println("Invalid UUID:", err)
			return []packet.LogPacket{}
		}
		scanner = sm.tsdb.Query(`SELECT id, user_id, data, time FROM storage WHERE point_id = ? and id < ? limit ? ALLOW FILTERING`, pointId, uuid, count).WithContext(ctx).Iter().Scanner()
	}
	logs := []packet.LogPacket{}
	for scanner.Next() {
		var id gocql.UUID
		var userId string
		var data string
		var timeVal int64
		err := scanner.Scan(&id, &userId, &data, &timeVal)
		if err != nil {
			log.Fatal(err)
		}
		logs = append(logs, packet.LogPacket{Id: id.String(), UserId: userId, Data: data, PointId: pointId, Time: timeVal})
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

func NewStorage(core core.ICore, storageRoot string, baseDbPath string, logsDbPath string) *StorageManager {
	log.Println("connecting to database...")
	os.MkdirAll(baseDbPath, os.ModePerm)
	kvdb, err := badger.Open(badger.DefaultOptions(baseDbPath).WithSyncWrites(true).WithTruncate(true))
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
	return &StorageManager{core: core, tsdb: session, kvdb: kvdb, storageRoot: storageRoot}
}
