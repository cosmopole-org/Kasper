package adapters

import (
	"encoding/json"
	"kasper/src/abstract"
	"log"
	"strings"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IStorage interface {
	Db() *gorm.DB
	AutoMigrate(...interface{}) error
	StorageRoot() string
	DoTrx(fc func(ITrx) error) (err error)
}

type TrxOptions struct {
	Reset bool
}

type ITrx interface {
	Db() *gorm.DB
	Mem() ICacheTrx
	Updates() []abstract.Update
	ClearError()
}

func BuildJsonFetcher(column string, pathDot string) string {
	// return "JSON_EXTRACT(`" + column + "`, '$." + pathDot + "')"
	return "json_extract_path_text(\"" + column + "\"::json, '" + strings.Join(strings.Split(pathDot, "."), "','") + "')"
}

func extractDbHolder(dbHolder any) *gorm.DB {
	var db *gorm.DB
	builder, ok := dbHolder.(func() *gorm.DB)
	if ok {
		db = builder()
	} else {
		db = dbHolder.(*gorm.DB)
	}
	db.Error = nil
	return db
}

func UpdateJson(dbHolder any, entity any, column string, pathDot string, v any) error {
	var value any
	if _, ok := v.(int16); ok {
		value = v
	} else if _, ok := v.(int32); ok {
		value = v
	} else if _, ok := v.(int64); ok {
		value = v
	} else if _, ok := v.(float32); ok {
		value = v
	} else if _, ok := v.(float64); ok {
		value = v
	} else if _, ok := v.(string); ok {
		value = v
	} else if _, ok := v.(bool); ok {
		value = v
	} else {
		va, err := json.Marshal(v)
		if err != nil {
			return err
		}
		value = datatypes.JSON(va)
	}
	if pathDot == "" {
		return extractDbHolder(dbHolder).UpdateColumn(column, value).Error
	}
	pathParts := strings.Split(pathDot, ".")
	// path := strings.Join(pathParts, ".")
	path := "{" + strings.Join(pathParts, ",") + "}"
	var finalValue = value
	if entity == nil {
		return extractDbHolder(dbHolder).UpdateColumn(column, datatypes.JSONSet(column).Set(path, finalValue)).Error
	}
	for i := 1; i < len(pathParts); i++ {
		err := extractDbHolder(dbHolder).First(entity, datatypes.JSONQuery(column).HasKey(pathParts[0:i]...)).Error
		if err != nil {
			log.Println(err)
			// path = strings.Join(pathParts[0:i], ".")
			path = "{" + strings.Join(pathParts[0:i], ",") + "}"
			root := map[string]interface{}{}
			rootTemp := root
			for j := i; j < len(pathParts)-1; j++ {
				next := map[string]interface{}{}
				rootTemp[pathParts[j]] = next
				rootTemp = next
			}
			rootTemp[pathParts[len(pathParts)-1]] = value
			finalValue = root
			break
		} else {
			finalValue = value
		}
	}
	return extractDbHolder(dbHolder).UpdateColumn(column, datatypes.JSONSet(column).Set(path, finalValue)).Error
}
