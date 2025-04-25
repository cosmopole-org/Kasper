package tool_storage

import (
	"context"
	"encoding/binary"
	"fmt"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/packet"
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

func (sm *StorageManager) LogTimeSieries(pointId string, userId string, data string) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	if err := sm.tsdb.Query(`INSERT INTO storage(id, point_id, user_id, data) VALUES (?, ?, ?)`,
		gocql.TimeUUID(), pointId, userId, data).WithContext(ctx).Exec(); err != nil {
		log.Fatal(err)
	}
}

func (sm *StorageManager) ReadPointLogs(pointId string) []packet.LogPacket {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	ctx := context.Background()
	scanner := sm.tsdb.Query(`SELECT id, user_id, data FROM storage WHERE point_id = ?`, pointId).WithContext(ctx).Iter().Scanner()
	var err error
	logs := []packet.LogPacket{}
	for scanner.Next() {
		var id gocql.UUID
		var userId string
		var data string
			err = scanner.Scan(&id, &userId, &data)
		if err != nil {
			log.Fatal(err)
		}
		logs = append(logs, packet.LogPacket{Id: id.String(), UserId: userId, Data: data})
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return logs
}

func (sm *StorageManager) GenId(origin string) string {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	trx := sm.kvdb.NewTransaction(true)
	defer trx.Commit()
	if origin == "global" {
		item, err := trx.Get([]byte("globalIdCounter"))
		var counter int64 = 0
		if err != nil {
			counter = 0
		}
		var b []byte
		item.Value(func(val []byte) error {
			b = val
			return nil
		})
		counter = int64(binary.BigEndian.Uint64(b))
		counter++
		nextB := [8]byte{}
		binary.BigEndian.PutUint64(nextB[:], uint64(counter))
		trx.Set([]byte("globalIdCounter"), nextB[:])
		return fmt.Sprintf("%d@%s", counter, origin)
	} else {
		item, err := trx.Get([]byte("localIdCounter"))
		var counter int64 = 0
		if err != nil {
			counter = 0
		}
		var b []byte
		item.Value(func(val []byte) error {
			b = val
			return nil
		})
		counter = int64(binary.BigEndian.Uint64(b))
		counter++
		nextB := [8]byte{}
		binary.BigEndian.PutUint64(nextB[:], uint64(counter))
		trx.Set([]byte("localIdCounter"), nextB[:])
		return fmt.Sprintf("%d@%s", counter, origin)
	}
}

func NewStorage(core core.ICore, storageRoot string) *StorageManager {
	log.Println("connecting to database...")
	kvdb, err := badger.Open(badger.DefaultOptions(os.Getenv("BASE_DB_PATH")).WithSyncWrites(true))
	if err != nil {
		panic(err)
	}
	cluster := gocql.NewCluster("localhost:9042")
	cluster.Keyspace = "example"
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()
	if err != nil {
		panic(err)
	}
	return &StorageManager{core: core, tsdb: session, kvdb: kvdb, storageRoot: storageRoot}
}
