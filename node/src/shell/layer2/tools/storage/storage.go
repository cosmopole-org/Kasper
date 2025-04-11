package tool_storage

import (
	// "encoding/json"
	"context"
	"kasper/src/abstract"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/adapters"
	tool_cache "kasper/src/shell/layer2/tools/cache"
	"log"
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type StorageManager struct {
	core        abstract.ICore
	logger      *modulelogger.Logger
	storageRoot string
	db          *gorm.DB
	cache       adapters.ICache
}

type TrxWrapper struct {
	core    abstract.ICore
	db      *gorm.DB
	mem     *tool_cache.CacheTrx
	Changes []abstract.Update
}

func (sm *StorageManager) StorageRoot() string {
	return sm.storageRoot
}
func (sm *StorageManager) Db() *gorm.DB {
	return sm.db
}
func (sm *StorageManager) AutoMigrate(args ...interface{}) error {
	return sm.db.AutoMigrate(args...)
}

type Recorder struct {
	Updates []abstract.Update
	Tw      *TrxWrapper
}

func (r Recorder) LogMode(logger.LogLevel) logger.Interface {
	return r
}
func (r Recorder) Info(context.Context, string, ...interface{}) {

}
func (r Recorder) Warn(context.Context, string, ...interface{}) {

}
func (r Recorder) Error(context.Context, string, ...interface{}) {

}
func (r Recorder) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, _ := fc()
	r.Tw.Changes = append(r.Tw.Changes, abstract.Update{Data: sql})
}

func (sm *StorageManager) DoTrx(fc func(control adapters.ITrx) error) (err error) {
	tw := &TrxWrapper{core: sm.core, Changes: []abstract.Update{}, mem: sm.cache.DoCacheTrx().(*tool_cache.CacheTrx)}
	return sm.db.Transaction(func(tx *gorm.DB) error {
		tx.Logger = Recorder{Tw: tw}
		tw.db = tx
		return fc(tw)
	})
}
func (tw *TrxWrapper) Mem() adapters.ICacheTrx {
	return tw.mem
}
func (tw *TrxWrapper) Db() *gorm.DB {
	return tw.db
}
func (tw *TrxWrapper) Updates() []abstract.Update {
	return tw.Changes

}
func (tw *TrxWrapper) ClearError() {
	tw.db.Error = nil
}

func NewStorage(core abstract.ICore, cache adapters.ICache, logger2 *modulelogger.Logger, storageRoot string, dialector gorm.Dialector) *StorageManager {
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
	db.AutoMigrate(&abstract.Counter{})
	db.Create(&abstract.Counter{Id: "globalIdCounter", Value: 0})
	return &StorageManager{core: core, db: db, storageRoot: storageRoot, logger: logger2, cache: cache}
}
