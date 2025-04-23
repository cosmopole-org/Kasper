package tool_storage

import (
	"encoding/binary"
	"fmt"
	"kasper/src/abstract/models/core"
	modulelogger "kasper/src/core/module/logger"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type StorageManager struct {
	core        core.ICore
	logger      *modulelogger.Logger
	storageRoot string
	sqldb       *gorm.DB
	kvdb        *badger.DB
	lock        sync.Mutex
}

func (sm *StorageManager) StorageRoot() string {
	return sm.storageRoot
}
func (sm *StorageManager) SqlDb() *gorm.DB {
	return sm.sqldb
}
func (sm *StorageManager) KvDb() *badger.DB {
	return sm.kvdb
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

func NewStorage(core core.ICore, logger2 *modulelogger.Logger, storageRoot string, dialector gorm.Dialector) *StorageManager {
	logger2.Println("connecting to database...")
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // tools writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,        // Don't include params in the SQL log
			Colorful:                  false,       // Disable color
		},
	)
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect database")
	}
	kvdb, err := badger.Open(badger.DefaultOptions(os.Getenv("BASE_DB_PATH")).WithSyncWrites(true))
	if err != nil {
		panic(err)
	}
	return &StorageManager{core: core, sqldb: db, kvdb: kvdb, storageRoot: storageRoot, logger: logger2}
}
