package tool_cache

import (
	"fmt"
	"kasper/cmd/babble/sigma/abstract"
	modulelogger "kasper/cmd/babble/sigma/core/module/logger"
	"kasper/cmd/babble/sigma/layer1/adapters"
	"kasper/cmd/babble/sigma/utils/crypto"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CacheTrx struct {
	Lock    sync.RWMutex
	Cache   adapters.ICache
	Changes []abstract.CacheUpdate
}

type Cache struct {
	Lock        sync.RWMutex
	Core        abstract.ICore
	logger      *modulelogger.Logger
	Indexes     map[string]map[string]string
	Dict        map[string]string
	RedisClient *redis.Client
}

func NewCache(core abstract.ICore, logger *modulelogger.Logger, redisUri string) *Cache {
	logger.Println("connecting to cache...")
	opts, err := redis.ParseURL(redisUri)
	if err != nil {
		panic(err)
	}
	client := redis.NewClient(opts)
	return &Cache{Core: core, RedisClient: client, Dict: map[string]string{}, Indexes: map[string]map[string]string{}, logger: logger}
}

func (m *Cache) DoCacheTrx() adapters.ICacheTrx {
	return &CacheTrx{Cache: m, Changes: []abstract.CacheUpdate{}}
}

func (m *Cache) Infra() any {
	return m.RedisClient
}

func (m *Cache) Put(key string, value string) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Dict[key] = value
	if strings.HasPrefix(key, "member::") {
		parts := strings.Split(key, "::")
		if len(parts) == 4 {
			indexKey := parts[0] + "::" + parts[1] + "::" + parts[2] + "::*"
			if m.Indexes[indexKey] != nil {
				m.Indexes[indexKey][key] = value
			} else {
				index := map[string]string{}
				index[key] = value
				m.Indexes[indexKey] = index
			}
		}
	}
}

func (m *Cache) Get(key string) string {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	val, ok := m.Dict[key]
	if ok {
		return val
	} else {
		return ""
	}
}

func (m *Cache) Keys(mainPart string) []string {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	val, ok := m.Indexes[mainPart]
	if ok {
		v := make([]string, 0, len(val))
		for key := range val {
			v = append(v, key)
		}
		return v
	} else {
		return []string{}
	}
}

func (m *Cache) Del(key string) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	delete(m.Dict, key)
}

func (m *Cache) GenId(db *gorm.DB, origin string) string {
	if origin == "global" {
		m.Lock.Lock()
		defer m.Lock.Unlock()
		val := abstract.Counter{Id: "globalIdCounter"}
		db.First(&val)
		val.Value = val.Value + 1
		db.Save(&val)
		return fmt.Sprintf("%d@%s", val.Value, origin)
	} else {
		return crypto.SecureUniqueId(m.Core.Id())
	}
}

func (m *CacheTrx) Put(key string, value string) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Changes = append(m.Changes, adapters.NewUpdatePut(key, value))
	m.Cache.Put(key, value)
}

func (m *CacheTrx) Del(key string) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Changes = append(m.Changes, adapters.NewUpdateDel(key))
	m.Cache.Del(key)
}

func (m *CacheTrx) Updates() []abstract.CacheUpdate {
	return m.Changes
}