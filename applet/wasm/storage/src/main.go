package main

import (
	model "applet/src/models"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

//go:module env
//export httpPost
func httpPost(k int32, kl int32, v int32, lv int32, p int32, pv int32) int64

//go:module env
//export plantTrigger
func plantTrigger(k int32, kl int32, v int32, lv int32, p int32, pv int32, count int32) int64

//go:module env
//export signalPoint
func signalPoint(k int32, kl int32, v int32, lv int32, p int32, pv int32, c int32, cv int32) int64

//go:module env
//export runDocker
func runDocker(k int32, kl int32, v int32, lv int32, c int32, cv int32) int64

//go:module env
//export execDocker
func execDocker(k int32, kl int32, v int32, lv int32, c int32, cv int32) int64

//go:module env
//export copyToDocker
func copyToDocker(k int32, kl int32, v int32, lv int32, c int32, cv int32, con int32, conv int32) int64

//go:module env
//export put
func put(k int32, kl int32, v int32, lv int32) int32

//go:module env
//export del
func del(k int32, kl int32) int32

//go:module env
//export get
func get(k int32, kl int32) int64

//go:module env
//export getByPrefix
func getByPrefix(k int32, kl int32) int64

//go:module env
//export consoleLog
func consoleLog(k int32, kl int32) int32

//go:module env
//export submitOnchainTrx
func submitOnchainTrx(tmO int32, tmL int32, keyO int32, keyL int32, inputO int32, inputL int32, metaO int32, metaL int32) int64

//go:module env
//export output
func output(k int32, kl int32) int32

//go:module env
//export newSyncTask
func newSyncTask(k int32, kl int32) int32

func bytesToPointer(d []byte) (int32, int32) {
	p := int32(uintptr(unsafe.Pointer(&(d[0]))))
	l := int32(len(d))
	return p, l
}

func pointerToBytes(p int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(p))
	valO := int32(binary.BigEndian.Uint32(b[:4]))
	valL := int32(binary.BigEndian.Uint32(b[4:]))
	bytes := []byte{}
	pointer := uintptr(valO)
	for nth := 0; nth < int(valL); nth++ {
		s := *(*int32)(unsafe.Pointer(pointer + uintptr(nth)))
		bytes = append(bytes, (byte(s)))
	}
	return bytes
}

func bytesToSinglePointer(data []byte) int64 {
	p, l := bytesToPointer(data)
	pb := make([]byte, 4)
	lb := make([]byte, 4)
	binary.BigEndian.PutUint32(pb, uint32(p))
	binary.BigEndian.PutUint32(lb, uint32(l))
	b := append(pb, lb...)
	result := int64(binary.BigEndian.Uint64(b))
	return result
}

// ---------------------------------------------------------------------------

type Logger struct{}

func (*Logger) Log(text string) {
	consoleLog(bytesToPointer([]byte(text)))
}

type Field struct {
	Type  string
	Value any
}
type EntityType[T any] struct {
	Id    string
	Props map[string]Field
	Store *EntityGroup[T]
}

var goToDbTypes = map[string]string{
	"int":     "i32",
	"int16":   "i16",
	"int32":   "i32",
	"int64":   "i64",
	"float32": "f32",
	"float64": "f64",
	"bool":    "bool",
	"string":  "str",
}

var defDbVals = map[string]any{
	"i16":  0,
	"i32":  0,
	"i64":  0,
	"f32":  0,
	"f64":  0,
	"bool": false,
	"str":  "",
	"list": nil,
}

var converters = map[string]func(reflect.Value) any{
	"i16": func(val reflect.Value) any {
		return int16(val.Int())
	},
	"i32": func(val reflect.Value) any {
		return int32(val.Int())
	},
	"i64": func(val reflect.Value) any {
		return int64(val.Int())
	},
	"f32": func(val reflect.Value) any {
		return float32(val.Float())
	},
	"f64": func(val reflect.Value) any {
		return float64(val.Float())
	},
	"bool": func(val reflect.Value) any {
		return val.Bool()
	},
	"str": func(val reflect.Value) any {
		return val.String()
	},
}

func NewEntityType[T any](db *Db, s T) *EntityType[T] {
	t := reflect.TypeOf(s)
	propsMap := map[string]Field{}
	for i := range t.NumField() {
		name := t.Field(i).Name
		typ := t.Field(i).Type.String()
		logger.Log(typ)
		dbTyp, ok := goToDbTypes[typ]
		if ok {
			typ = dbTyp
		} else {
			typ = "list::" + name[:len(name)-1]
		}
		propsMap[name] = Field{Type: typ, Value: defDbVals[typ]}
	}
	id := t.Name()
	et := &EntityType[T]{
		Id:    id,
		Props: propsMap,
		Store: nil,
	}
	et.Store = &EntityGroup[T]{
		Prefix:     id,
		EntityType: et,
		Db:         db,
		Map:        map[string]*Entity{},
	}
	logger.Log("id: " + id)
	return et
}
func (et *EntityType[T]) NewEntity(s *T) *Entity {
	t := reflect.TypeOf(*s)
	r := reflect.ValueOf(s)
	logger.Log("hello 2")
	id := r.Elem().FieldByName("Id").String()
	logger.Log("hello 3")
	props := map[string]any{}
	for i := range t.NumField() {
		name := t.Field(i).Name
		logger.Log("hello 4")
		typ := et.Props[name].Type
		f := r.Elem().FieldByName(name)
		logger.Log("hello 5")
		if strings.Split(typ, "::")[0] == "list" {
			if f.IsValid() {
				if f.CanSet() {
					logger.Log("hello 6")
					if strings.Split(typ, "::")[0] == "list" {
						logger.Log("hello 7 " + typ)
						val := InstantiateEntityGroup(et.Store.Db, strings.Split(typ, "::")[1], et.Id+"::"+id+"::"+name)
						logger.Log("hello 8")
						//props[name] = val
						r.Elem().FieldByName(name).Set(reflect.ValueOf(val))
					}
				}
			}
		}
		c, ok := converters[typ]
		if ok {
			logger.Log("found " + typ + " " + name)
			value := c(f)
			props[name] = value
		} else {
			logger.Log("not found " + typ + " " + name)
		}
	}
	e := &Entity{Props: map[string]any{}}
	e.Id = id
	for k, v := range et.Props {
		if nv, ok := props[k]; ok {
			e.Props[k] = nv
		} else {
			e.Props[k] = v.Value
		}
	}
	return e
}

type Entity struct {
	Id    string
	Props map[string]any
}

func (e *Entity) GetProp(propName string) any {
	return e.Props[propName]
}

type EntityIndex struct {
	FieldName string
	IndexType string
	Store     []*Entity
}

type EntityGroup[T any] struct {
	Prefix     string
	EntityType *EntityType[T]
	Db         *Db
	Map        map[string]*Entity
	Indexes    map[string]*EntityIndex
}

func NewEntityGroup[T any](prefix string, et *EntityType[T], db *Db) *EntityGroup[T] {
	logger.Log("hello 12")
	return &EntityGroup[T]{
		Prefix:     prefix,
		EntityType: et,
		Db:         db,
		Map:        map[string]*Entity{},
	}
}

func mapToStruct[T any](db *Db, et *EntityType[T], src map[string]any) T {
	logger.Log("ok 0")
	destP := new(T)
	ps := reflect.ValueOf(destP)
	s := ps.Elem()
	if s.Kind() == reflect.Struct {
		id := src["Id"].(string)
		for k, v := range et.Props {
			f := s.FieldByName(k)
			if f.IsValid() {
				if f.CanSet() {
					if strings.Split(v.Type, "::")[0] == "list" {
						logger.Log("ok 1 " + v.Type + " " + et.Id)
						f.Set(reflect.ValueOf(InstantiateEntityGroup(db, strings.Split(v.Type, "::")[1], et.Id+"::"+id+"::"+k)))
						logger.Log("ok 2")
					} else if v.Type == "i16" {
						var x int64 = 0
						x1, ok1 := src[k].(int16)
						x2, ok2 := src[k].(float64)
						if ok1 {
							x = int64(x1)
						} else if ok2 {
							x = int64(x2)
						}
						if !f.OverflowInt(x) {
							f.SetInt(x)
						}
					} else if v.Type == "i32" {
						var x int64 = 0
						x1, ok1 := src[k].(int32)
						x2, ok2 := src[k].(float64)
						if ok1 {
							x = int64(x1)
						} else if ok2 {
							x = int64(x2)
						}
						if !f.OverflowInt(x) {
							f.SetInt(x)
						}
					} else if v.Type == "i64" {
						var x int64 = 0
						x1, ok1 := src[k].(int64)
						x2, ok2 := src[k].(float64)
						if ok1 {
							x = int64(x1)
						} else if ok2 {
							x = int64(x2)
						}
						if !f.OverflowInt(x) {
							f.SetInt(x)
						}
					} else if v.Type == "f32" {
						var x float64 = 0
						x1, ok1 := src[k].(float32)
						x2, ok2 := src[k].(float64)
						if ok1 {
							x = float64(x1)
						} else if ok2 {
							x = float64(x2)
						}
						if !f.OverflowFloat(x) {
							f.SetFloat(x)
						}
					} else if v.Type == "f64" {
						x := float64(src[k].(float64))
						if !f.OverflowFloat(x) {
							f.SetFloat(x)
						}
					} else if v.Type == "bool" {
						x := src[k].(bool)
						f.SetBool(x)
					} else if v.Type == "str" {
						x := src[k].(string)
						f.SetString(x)
					}
				}
			}
		}
	}
	return *destP
}

func (eg *EntityGroup[T]) InsertEntity(e *Entity) {
	logger.Log("hello 13")
	b, _ := json.Marshal(e.Props)
	logger.Log("hello 14 " + string(b))
	eg.Db.Put("table::"+eg.EntityType.Id+"::"+e.Id, b)
	logger.Log("hello 15")
	if eg.EntityType.Id != eg.Prefix {
		logger.Log("hello 16 [" + eg.EntityType.Id + "::" + e.Id + "]")
		eg.Db.Put(eg.Prefix+"::"+e.Id, []byte(eg.EntityType.Id+"::"+e.Id))
	}
	logger.Log("hello 17")
	eg.Map[e.Id] = e
}
func (eg *EntityGroup[T]) CreateAndInsert(s *T) {
	logger.Log("hello 1")
	eg.InsertEntity(eg.EntityType.NewEntity(s))
	logger.Log("hello 18")
}
func (eg *EntityGroup[T]) DeleteById(id string) {
	if eg.Prefix == eg.EntityType.Id {
		eg.Db.Del("table::" + eg.Prefix + "::" + id)
	} else {
		eg.Db.Del(eg.Prefix + "::" + id)
	}
}
func (eg *EntityGroup[T]) FindById(id string) T {
	var b []byte
	logger.Log("ok 0.1")
	if eg.Prefix == eg.EntityType.Id {
		logger.Log("ok 0.2")
		b = eg.Db.Get("table::" + eg.Prefix + "::" + id)
	} else {
		logger.Log("ok 0.3")
		b = eg.Db.Get(eg.Prefix + "::" + id)
	}
	logger.Log("ok 0.4")
	if len(b) == 0 {
		logger.Log("ok 0.5")
		return *new(T)
	}
	logger.Log("ok 0.6 " + string(b))
	src := map[string]any{}
	json.Unmarshal(b, &src)
	logger.Log("ok 0.7")
	for k, v := range eg.EntityType.Props {
		logger.Log("ok 0.8 " + v.Type)
		if v.Type == "i16" {
			var x int16 = 0
			x1, ok1 := src[k].(int16)
			x2, ok2 := src[k].(float64)
			if ok1 {
				x = int16(x1)
			} else if ok2 {
				x = int16(x2)
			}
			src[k] = x
		} else if v.Type == "i32" {
			var x int32 = 0
			x1, ok1 := src[k].(int32)
			x2, ok2 := src[k].(float64)
			if ok1 {
				x = int32(x1)
			} else if ok2 {
				x = int32(x2)
			}
			src[k] = x
		} else if v.Type == "i64" {
			var x int64 = 0
			x1, ok1 := src[k].(int64)
			x2, ok2 := src[k].(float64)
			if ok1 {
				x = int64(x1)
			} else if ok2 {
				x = int64(x2)
			}
			src[k] = x
		} else if v.Type == "f32" {
			var x float32 = 0
			x1, ok1 := src[k].(float32)
			x2, ok2 := src[k].(float64)
			if ok1 {
				x = float32(x1)
			} else if ok2 {
				x = float32(x2)
			}
			src[k] = x
		} else if v.Type == "f64" {
			x := float64(src[k].(float64))
			src[k] = x
		} else if v.Type == "bool" {
			x := src[k].(bool)
			src[k] = x
		} else if v.Type == "str" {
			logger.Log(reflect.TypeOf(src[k]).Name())
			x := src[k].(string)
			src[k] = x
		}
	}
	k := src["Id"].(string)
	eg.Map[k] = &Entity{Id: k, Props: src}
	return mapToStruct(eg.Db, eg.EntityType, src)
}
func (eg *EntityGroup[T]) Load() {
	bs := [][]byte{}
	if eg.Prefix == eg.EntityType.Id {
		bs = eg.Db.GetByPrefix("table::" + eg.Prefix)
		for _, b := range bs {
			logger.Log(string(b))
		}
	} else {
		logger.Log("hello 16 [" + eg.Prefix + "]")
		keys := eg.Db.GetByPrefix(eg.Prefix)
		for _, keyB := range keys {
			key := string(keyB)
			if eg.Db == nil {
				logger.Log(key + " false")
			} else {
				logger.Log(key + " true")
			}
			b := eg.Db.Get("table::" + key)
			logger.Log(string(b))
			bs = append(bs, b)
		}
	}
	for _, b := range bs {
		src := map[string]any{}
		json.Unmarshal(b, &src)
		for k, v := range eg.EntityType.Props {
			if v.Type == "i16" {
				var x int16 = 0
				x1, ok1 := src[k].(int16)
				x2, ok2 := src[k].(float64)
				if ok1 {
					x = int16(x1)
				} else if ok2 {
					x = int16(x2)
				}
				src[k] = x
			} else if v.Type == "i32" {
				var x int32 = 0
				x1, ok1 := src[k].(int32)
				x2, ok2 := src[k].(float64)
				if ok1 {
					x = int32(x1)
				} else if ok2 {
					x = int32(x2)
				}
				src[k] = x
			} else if v.Type == "i64" {
				var x int64 = 0
				x1, ok1 := src[k].(int64)
				x2, ok2 := src[k].(float64)
				if ok1 {
					x = int64(x1)
				} else if ok2 {
					x = int64(x2)
				}
				src[k] = x
			} else if v.Type == "f32" {
				var x float32 = 0
				x1, ok1 := src[k].(float32)
				x2, ok2 := src[k].(float64)
				if ok1 {
					x = float32(x1)
				} else if ok2 {
					x = float32(x2)
				}
				src[k] = x
			} else if v.Type == "f64" {
				x := float64(src[k].(float64))
				src[k] = x
			} else if v.Type == "bool" {
				x := src[k].(bool)
				src[k] = x
			} else if v.Type == "str" {
				x := src[k].(string)
				src[k] = x
			}
		}
		k := src["Id"].(string)
		eg.Map[k] = &Entity{Id: k, Props: src}
	}
}
func (eg *EntityGroup[T]) Read(filterBy string, sortBy string, sortOrder string) []T {
	eg.Load()
	list := []*Entity{}
	for _, v := range eg.Map {
		list = append(list, v)
	}
	if len(list) == 0 {
		return []T{}
	}
	if sortBy != "" {
		if sortOrder == "desc" {
			if _, ok := list[0].Props[sortBy].(int16); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(int16) > list[j].Props[sortBy].(int16)
				})
			} else if _, ok := list[0].Props[sortBy].(int16); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(int32) > list[j].Props[sortBy].(int32)
				})
			} else if _, ok := list[0].Props[sortBy].(int64); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(int64) > list[j].Props[sortBy].(int64)
				})
			} else if _, ok := list[0].Props[sortBy].(float32); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(float32) > list[j].Props[sortBy].(float32)
				})
			} else if _, ok := list[0].Props[sortBy].(float64); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(float64) > list[j].Props[sortBy].(float64)
				})
			}
		} else {
			if _, ok := list[0].Props[sortBy].(int16); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(int16) < list[j].Props[sortBy].(int16)
				})
			} else if _, ok := list[0].Props[sortBy].(int16); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(int32) < list[j].Props[sortBy].(int32)
				})
			} else if _, ok := list[0].Props[sortBy].(int64); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(int64) < list[j].Props[sortBy].(int64)
				})
			} else if _, ok := list[0].Props[sortBy].(float32); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(float32) < list[j].Props[sortBy].(float32)
				})
			} else if _, ok := list[0].Props[sortBy].(float64); ok {
				sort.Slice(list, func(i, j int) bool {
					return list[i].Props[sortBy].(float64) < list[j].Props[sortBy].(float64)
				})
			}
		}
	}
	result := []T{}
	for _, item := range list {
		typedItem := mapToStruct(eg.Db, eg.EntityType, item.Props)
		result = append(result, typedItem)
	}
	return result
}

type Db struct {
	EntityTypes map[string]any
}

func (*Db) Put(key string, value []byte) {
	kP, kL := bytesToPointer([]byte(key))
	vP, vL := bytesToPointer(value)
	put(kP, kL, vP, vL)
}
func (*Db) Del(key string) {
	kP, kL := bytesToPointer([]byte(key))
	del(kP, kL)
}
func (*Db) Get(key string) []byte {
	kP, kL := bytesToPointer([]byte(key))
	logger.Log("step 100 " + key)
	val := get(kP, kL)
	s := strconv.FormatInt(val, 10)
	logger.Log("step 101 " + s)
	return pointerToBytes(val)
}
func (*Db) GetByPrefix(key string) [][]byte {
	kP, kL := bytesToPointer([]byte(key))
	val := getByPrefix(kP, kL)
	data := pointerToBytes(val)
	logger.Log("first data: " + string(data))
	type BytesInBytes struct {
		Data []string `json:"data"`
	}
	arr := BytesInBytes{Data: []string{}}
	err := json.Unmarshal(data, &arr)
	if err != nil {
		logger.Log(err.Error())
	}
	result := [][]byte{}
	for _, b := range arr.Data {
		logger.Log("first data: " + string(b))
		result = append(result, []byte(b))
	}
	return result
}

type Chain struct {
}

func (c *Chain) SubmitAppletPacketTrx(pointId string, targetMachineId string, key string, userId string, signature string, tokenId string, tag string, input any) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	tagO, tagL := bytesToPointer([]byte("00" + tag))
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key + "|" + userId + "|" + signature + "|" + tokenId))
	b, e := json.Marshal(input)
	if e != nil {
		logger.Log(e.Error())
		return []byte("{}")
	}
	inputO, inputL := bytesToPointer(b)
	resP := submitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitAppletFileTrx(pointId string, targetMachineId string, fileId string, userId string, signature string, tokenId string, tag string) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	tagO, tagL := bytesToPointer([]byte("10" + tag))
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + "/storage/upload" + "|" + userId + "|" + signature + "|" + tokenId))
	inputO, inputL := bytesToPointer([]byte(fileId))
	resP := submitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitBasePacketTrx(pointId string, key string, userId string, signature string, tag string, input []byte) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key + "|" + userId + "|" + signature + "|" + "-"))
	tagO, tagL := bytesToPointer([]byte("01" + tag))
	b, e := json.Marshal(input)
	if e != nil {
		logger.Log(e.Error())
		return []byte("{}")
	}
	inputO, inputL := bytesToPointer(b)
	resP := submitOnchainTrx(0, 0, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitBaseFileTrx(pointId string, fileId string, userId string, signature string, tag string) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + "/storage/upload" + "|" + userId + "|" + signature + "|" + "-"))
	tagO, tagL := bytesToPointer([]byte("11" + tag))
	inputO, inputL := bytesToPointer([]byte(fileId))
	resP := submitOnchainTrx(0, 0, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) PlantTrigger(count int32, pointId string, tag string, input map[string]any) {
	tagO, tagL := bytesToPointer([]byte(tag))
	piO, piL := bytesToPointer([]byte(pointId))
	b, _ := json.Marshal(input)
	inputO, inputL := bytesToPointer(b)
	plantTrigger(tagO, tagL, inputO, inputL, piO, piL, count)
}

type Trx[T any] struct {
	Db    *T
	Chain *Chain
}

func ParseArgs(a int64) model.Send {
	input := model.Send{}
	str := pointerToBytes(a)
	logger.Log(string(str))
	e := json.Unmarshal(str, &input)
	if e != nil {
		logger.Log("unable to parse args as send.")
		return model.Send{}
	}
	return input
}

type Vm struct{}

func (vm *Vm) RunDocker(imageName string, containerName string, inputFiles map[string]string) *model.File {
	kp, kl := bytesToPointer([]byte(imageName))
	cp, cl := bytesToPointer([]byte(containerName))
	b, _ := json.Marshal(inputFiles)
	vp, vl := bytesToPointer(b)
	res := pointerToBytes(runDocker(kp, kl, vp, vl, cp, cl))
	file := &model.File{}
	err := json.Unmarshal(res, file)
	if err != nil {
		logger.Log(err.Error())
	}
	return file
}

func (vm *Vm) ExecDocker(imageName string, containerName string, command string) string {
	kp, kl := bytesToPointer([]byte(imageName))
	cp, cl := bytesToPointer([]byte(containerName))
	cop, col := bytesToPointer([]byte(command))
	res := pointerToBytes(execDocker(kp, kl, cp, cl, cop, col))
	logger.Log(string(res))
	result := map[string]any{}
	json.Unmarshal(res, &result)
	return result["data"].(string)
}

func (vm *Vm) CopyToDocker(imageName string, containerName string, fileName string, content string) {
	kp, kl := bytesToPointer([]byte(imageName))
	cp, cl := bytesToPointer([]byte(containerName))
	cop, col := bytesToPointer([]byte(fileName))
	conp, conl := bytesToPointer([]byte(content))
	copyToDocker(kp, kl, cp, cl, cop, col, conp, conl)
}

func SendSignal(typ string, pointId string, userId string, data string, isTemp bool) {
	temp := "false"
	if isTemp {
		temp = "true"
	}
	kp, kl := bytesToPointer([]byte(typ + "|" + temp))
	cp, cl := bytesToPointer([]byte(pointId))
	cop, col := bytesToPointer([]byte(userId))
	cop2, col2 := bytesToPointer([]byte(data))
	signalPoint(kp, kl, cp, cl, cop, col, cop2, col2)
}

// ---------------------------------------------------------------------------

type MyDb struct {
	BaseDB *Db
	Users  *EntityGroup[User]
	Points *EntityGroup[Point]
	Docs   *EntityGroup[Doc]
}

type User struct {
	Id       string
	Name     string
	AuthCode string
}

type Point struct {
	Id         string
	IsPublic   bool
	LastUpdate int64
	Docs       *EntityGroup[Doc] `json:"-"`
}

type Doc struct {
	Id        string
	Title     string
	FileId    string
	MimeType  string
	Category  string
	CreatorId string
	PointId   string
}

func NewMyDb() *MyDb {
	db := &Db{
		EntityTypes: map[string]any{},
	}
	docsColl := NewEntityType(db, Doc{})
	db.EntityTypes[docsColl.Id] = docsColl
	pointsColl := NewEntityType(db, Point{})
	db.EntityTypes[pointsColl.Id] = pointsColl
	usersColl := NewEntityType(db, User{})
	db.EntityTypes[usersColl.Id] = usersColl
	return &MyDb{
		BaseDB: db,
		Docs:   docsColl.Store,
		Users:  usersColl.Store,
		Points: pointsColl.Store,
	}
}

func InstantiateEntityGroup(db *Db, id string, prefix string) any {
	if (id == reflect.TypeOf(User{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[User]), db)
	} else if (id == reflect.TypeOf(Doc{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[Doc]), db)
	} else if (id == reflect.TypeOf(Point{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[Point]), db)
	}
	return nil
}

type NetHttp struct{}

func (nh *NetHttp) Post(url string, headers map[string]string, body any) string {
	urlO, urlL := bytesToPointer([]byte(url))
	headersJson, _ := json.Marshal(headers)
	bodyJson, _ := json.Marshal(body)
	headersO, headersL := bytesToPointer(headersJson)
	bodyO, bodyL := bytesToPointer(bodyJson)
	return string(pointerToBytes(httpPost(urlO, urlL, headersO, headersL, bodyO, bodyL)))
}

var logger = &Logger{}

var network = &NetHttp{}

var syncTasks = map[string]func(){}

func doSync(task func(), deps []string, name string) {
	syncTasks[name] = task
	packet := map[string]any{
		"name": name,
		"deps": deps,
	}
	str, _ := json.Marshal(packet)
	p, l := bytesToPointer(str)
	newSyncTask(p, l)
}

// ---------------------------------------------------------------------------

//export runTask
func runTask(a int64) int32 {
	b := pointerToBytes(a)
	syncTasks[string(b)]()
	return 0
}

type TriggerCallbackPacket struct {
	Attachment string   `json:"attachment"`
	Payloads   []string `json:"payloads"`
}

type Attachment struct {
	Action   string            `json:"action"`
	SrcFiles map[string]string `json:"srcFiles"`
}

type File struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	PointId string `json:"pointId" gorm:"column:topic_id"`
	OwnerId string `json:"senderId" gorm:"column:sender_id"`
}

type FileResponse struct {
	File File `json:"file"`
}

func answer(pointId string, userId string, data any) {
	res, _ := json.Marshal(data)
	SendSignal("single", pointId, userId, string(res), true)
}

func broadcast(pointId string, data any) {
	res, _ := json.Marshal(data)
	SendSignal("broadcast", pointId, "-", string(res), true)
}

//export run
func run(a int64) int64 {

	goToDbTypes = map[string]string{
		"int":     "i32",
		"int16":   "i16",
		"int32":   "i32",
		"int64":   "i64",
		"float32": "f32",
		"float64": "f64",
		"bool":    "bool",
		"string":  "str",
	}

	defDbVals = map[string]any{
		"i16":  0,
		"i32":  0,
		"i64":  0,
		"f32":  0,
		"f64":  0,
		"bool": false,
		"str":  "",
		"list": nil,
	}

	converters = map[string]func(reflect.Value) any{
		"i16": func(val reflect.Value) any {
			return int16(val.Int())
		},
		"i32": func(val reflect.Value) any {
			return int32(val.Int())
		},
		"i64": func(val reflect.Value) any {
			return int64(val.Int())
		},
		"f32": func(val reflect.Value) any {
			return float32(val.Float())
		},
		"f64": func(val reflect.Value) any {
			return float64(val.Float())
		},
		"bool": func(val reflect.Value) any {
			return val.Bool()
		},
		"str": func(val reflect.Value) any {
			return val.String()
		},
	}

	logger.Log("parsing input...")
	input := map[string]any{}
	signal := ParseArgs(a)
	err := json.Unmarshal([]byte(signal.Data), &input)
	if err != nil {
		logger.Log(err.Error())
	}
	trx := &Trx[MyDb]{
		Db: NewMyDb(),
	}
	actRaw, ok := input["type"]
	if !ok {
		answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 1})
	}
	act, ok := actRaw.(string)
	if !ok {
		answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 2})
	}
	vm := Vm{}

	switch act {
	case "adminInit":
		{
			vm.RunDocker("storage", "storage", map[string]string{})
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "adminInitRes", "response": "initialized successfully"})
			break
		}
	case "createStorage":
		{
			logger.Log("step 1")
			trx.Db.Users.CreateAndInsert(&User{
				Id:       signal.User.Id,
				Name:     signal.User.Username,
				AuthCode: "",
			})
			logger.Log("step 2")
			redirectUrlRaw, ok := input["redirectUrl"]
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "initStorageRes", "success": false, "errCode": 3})
				return 0
			}
			logger.Log("step 3")
			redirectUrl, ok := redirectUrlRaw.(string)
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "initStorageRes", "success": false, "errCode": 4})
				return 0
			}
			logger.Log("step 4")
			res := vm.ExecDocker("storage", "storage", "/app/storage --command=createStorage --redirectUrl="+redirectUrl+" --userId="+signal.User.Id)
			logger.Log("step 5")
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "initStorageRes", "response": res})
			break
		}
	case "authorizeStorage":
		{
			authCodeRaw, ok := input["authCode"]
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "authStorageRes", "success": false, "errCode": 3})
				return 0
			}
			authCode, ok := authCodeRaw.(string)
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "authStorageRes", "success": false, "errCode": 4})
				return 0
			}
			trx.Db.Users.CreateAndInsert(&User{
				Id:       signal.User.Id,
				Name:     signal.User.Username,
				AuthCode: authCode,
			})
			res := vm.ExecDocker("storage", "storage", "/app/storage --command=authorizeStorage --userId="+signal.User.Id+" --authCode="+authCode)
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "authStorageRes", "response": res})
			break
		}
	case "listFiles":
		{
			res := vm.ExecDocker("storage", "storage", "/app/storage --command=listFiles --userId="+signal.User.Id)
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "listFilesRes", "response": res})
			break
		}
	case "upload":
		{
			trx.Db.Points.CreateAndInsert(&Point{
				Id:         signal.Point.Id,
				IsPublic:   signal.Point.IsPublic,
				LastUpdate: time.Now().UnixMilli(),
			})
			point := trx.Db.Points.FindById(signal.Point.Id)
			user := trx.Db.Users.FindById(signal.User.Id)
			if user.Id == "" {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 3})
				return 0
			}
			contentRaw := input["content"]
			if contentRaw == nil {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 4})
				return 0
			}
			content, ok := contentRaw.(string)
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 5})
				return 0
			}
			totalSizeRaw := input["totalSize"]
			if totalSizeRaw == nil {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 6})
				return 0
			}
			totalSize, ok := totalSizeRaw.(float64)
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 7})
				return 0
			}
			mimeTypeRaw := input["mimeType"]
			if mimeTypeRaw == nil {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 8})
				return 0
			}
			mimeType, ok := mimeTypeRaw.(string)
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "success": false, "errCode": 9})
				return 0
			}
			fileKey := ""
			if fkRaw, ok := input["fileKey"]; ok {
				if fk, ok := fkRaw.(string); ok {
					fileKey = fk
				}
			}
			vm.CopyToDocker("storage", "storage", fileKey, content)
			res := vm.ExecDocker("storage", "storage", "/app/storage --command=upload --userId="+signal.User.Id+" --fileContentType="+mimeType+" --file="+fileKey+" --totalSize="+fmt.Sprintf("%d", int(totalSize)))
			resObj := map[string]any{}
			res = "{" + strings.Join(strings.Split(res, "{")[1:], "{")
			resObj = map[string]any{}
			json.Unmarshal([]byte(res), &resObj)
			fileType := ""
			if strings.HasPrefix(mimeType, "image/") {
				fileType = "image"
			} else if strings.HasPrefix(mimeType, "audio/") {
				fileType = "audio"
			} else if strings.HasPrefix(mimeType, "video/") {
				fileType = "video"
			} else {
				fileType = "document"
			}
			doc := &Doc{
				Id:        strings.ReplaceAll(uuid.NewString(), "-", ""),
				FileId:    resObj["fileId"].(string),
				MimeType:  mimeType,
				Category:  fileType,
				CreatorId: signal.User.Id,
				PointId:   signal.Point.Id,
				Title:     fileKey,
			}
			point.Docs.Load()
			point.Docs.CreateAndInsert(doc)
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "uploadRes", "docId": doc.Id})
			broadcast(signal.Point.Id, map[string]any{"type": "uploadNotif", "docId": doc.Id})
			break
		}
	case "download":
		{
			user := trx.Db.Users.FindById(signal.User.Id)
			if user.Id == "" {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 3})
				return 0
			}
			var point Point
			pIdRaw := input["pointId"]
			if pIdRaw != nil {
				p := trx.Db.Points.FindById(pIdRaw.(string))
				if p.IsPublic == true {
					point = p
				}
			} else {
				point = trx.Db.Points.FindById(signal.Point.Id)
			}
			docIdRaw := input["docId"]
			if docIdRaw == nil {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 4})
				return 0
			}
			docId, ok := docIdRaw.(string)
			if !ok {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 5})
				return 0
			}
			doc := trx.Db.Docs.FindById(docId)
			if doc.PointId == point.Id {
				if doc.Id == "" {
					answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 6})
					return 0
				}
				res := vm.ExecDocker("storage", "storage", "/app/storage --command=download --userId="+doc.CreatorId+" --fileId="+doc.FileId)
				resObj := map[string]any{}
				res = "{" + strings.Join(strings.Split(res, "{")[1:], "{")
				resObj = map[string]any{}
				json.Unmarshal([]byte(res), &resObj)
				answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "downloadRes", "docPayload": resObj["data"], "doc": doc})
			}
			break
		}
	case "pointFiles":
		{
			user := trx.Db.Users.FindById(signal.User.Id)
			if user.Id == "" {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 3})
				return 0
			}
			p := trx.Db.Points.FindById(signal.Point.Id)
			if p.Id == "" {
				trx.Db.Points.CreateAndInsert(&Point{
					Id:         signal.Point.Id,
					IsPublic:   signal.Point.IsPublic,
					LastUpdate: 0,
				})
			}
			point := trx.Db.Points.FindById(signal.Point.Id)
			if point.Id == "" {
				answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 4})
				return 0
			}
			point.Docs.Load()
			docs := point.Docs.Read("all", "", "")
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "pointFilesRes", "docs": docs})
			break
		}
	case "listTopMedia":
		{
			points := trx.Db.Points.Read("all", "LastUpdate", "desc")
			step := 0
			docs := []Doc{}
			for _, point := range points {
				if step > 5 {
					break
				}
				if point.IsPublic {
					docs = append(docs, point.Docs.Read("all", "", "")...)
					step++
				}
			}
			answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "listTopMediaRes", "docs": docs})
		}
	}

	return 0
}

func main() {
	fmt.Println()
	fmt.Println("module starting...")
	fmt.Println()
}
