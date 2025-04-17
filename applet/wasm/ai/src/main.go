package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

//go:module env
//export runDocker
func runDocker(k int32, kl int32, v int32, lv int32, b int32, bl int32) int32

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
func submitOnchainTrx(tmO int32, tmL int32, keyO int32, keyL int32, inputO int32, inputL int32) int64

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
		dbTyp, ok := goToDbTypes[typ]
		if ok {
			typ = dbTyp
		} else {
			typ = "list::" + t.Field(i).Tag.Get("entity")
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
	return et
}
func (et *EntityType[T]) NewEntity(s *T) *Entity {
	t := reflect.TypeOf(*s)
	r := reflect.ValueOf(s)
	id := r.Elem().FieldByName("Id").String()
	props := map[string]any{}
	for i := range t.NumField() {
		name := t.Field(i).Name
		typ := et.Props[name].Type
		f := r.Elem().FieldByName(name)
		if strings.Split(typ, "::")[0] == "list" {
			if f.IsValid() {
				if f.CanSet() {
					if strings.Split(typ, "::")[0] == "list" {
						val := InstantiateEntityGroup(et.Store.Db, strings.Split(typ, "::")[1], et.Id+"::"+id+"::"+name)
						//props[name] = val
						r.Elem().FieldByName(name).Set(reflect.ValueOf(val))
					}
				}
			}
		}
		c, ok := converters[typ]
		if ok {
			value := c(f)
			props[name] = value
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
	return &EntityGroup[T]{
		Prefix:     prefix,
		EntityType: et,
		Db:         db,
		Map:        map[string]*Entity{},
	}
}

func mapToStruct[T any](db *Db, et *EntityType[T], src map[string]any) T {
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
						f.Set(reflect.ValueOf(InstantiateEntityGroup(db, strings.Split(v.Type, "::")[1], et.Id+"::"+id+"::"+k)))
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
	b, _ := json.Marshal(e.Props)
	eg.Db.Put("table::"+eg.EntityType.Id+"::"+e.Id, b)
	if eg.EntityType.Id != eg.Prefix {
		eg.Db.Put(eg.Prefix+"::"+e.Id, []byte(eg.EntityType.Id+"::"+e.Id))
	}
	eg.Map[e.Id] = e
}
func (eg *EntityGroup[T]) CreateAndInsert(s *T) {
	eg.InsertEntity(eg.EntityType.NewEntity(s))
}
func (eg *EntityGroup[T]) DeleteById(id string) {
	if eg.Prefix == eg.EntityType.Id {
		eg.Db.Del("table::" + eg.Prefix + "::" + id)
	} else {
		eg.Db.Del(eg.Prefix + "::" + id)
	}
}
func (eg *EntityGroup[T]) Load() {
	bs := [][]byte{}
	if eg.Prefix == eg.EntityType.Id {
		bs = eg.Db.GetByPrefix("table::" + eg.Prefix)
	} else {
		keys := eg.Db.GetByPrefix(eg.Prefix)
		for _, keyB := range keys {
			key := string(keyB)
			bs = append(bs, eg.Db.Get("table::"+key))
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
	if sortOrder == "desc" {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Props[sortBy].(int32) > list[j].Props[sortBy].(int32)
		})
	} else {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Props[sortBy].(int32) < list[j].Props[sortBy].(int32)
		})
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
	val := get(kP, kL)
	return pointerToBytes(val)
}
func (*Db) GetByPrefix(key string) [][]byte {
	kP, kL := bytesToPointer([]byte(key))
	val := getByPrefix(kP, kL)
	data := pointerToBytes(val)
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
		result = append(result, []byte(b))
	}
	return result
}

type Chain struct {
}

func (c *Chain) SubmitTrx(targetMachineId string, key string, input any) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	keyO, keyL := bytesToPointer([]byte(key))
	b, e := json.Marshal(input)
	if e != nil {
		logger.Log(e.Error())
		return []byte("{}")
	}
	inputO, inputL := bytesToPointer(b)
	resP := submitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitRawTrx(targetMachineId string, key string, input []byte) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	keyO, keyL := bytesToPointer([]byte(key))
	inputO, inputL := bytesToPointer(input)
	resP := submitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL)
	result := pointerToBytes(resP)
	return result
}

type Trx[T any] struct {
	Db    *T
	Chain *Chain
}

func ParseArgs(a int64) string {
	// input := model.Send{}
	args := string(pointerToBytes(a))
	logger.Log(args)
	if strings.Contains(args, "offchain") {
		return "offchain"
	} else {
		return "onchain"
	}
	// logger.Log(b))
	// // e := json.Unmarshal(b, &input)
	// if e != nil {
	// 	// logger.Log("unable to parse args as send.")
	// 	return ""
	// }
	// logger.Log(input.Data)
	// return input.Data
}

type Vm struct{}

func (vm *Vm) RunDocker(imageName string, inputFileId string, inputFileName string) {
	kp, kl := bytesToPointer([]byte(imageName))
	vp, vl := bytesToPointer([]byte(inputFileId))
	bp, bl := bytesToPointer([]byte(inputFileName))
	runDocker(kp, kl, vp, vl, bp, bl)
}

// ---------------------------------------------------------------------------

type MyDb struct {
	BaseDB *Db
	Users  *EntityGroup[User]
	Jobs   *EntityGroup[Job]
}

type User struct {
	Id   string
	Name string
	Age  int
}

type Job struct {
	Id     string
	Title  string
	Income int
	Users  *EntityGroup[User] `json:"-" entity:"User"`
}

func NewMyDb() *MyDb {
	db := &Db{
		EntityTypes: map[string]any{},
	}
	usersColl := NewEntityType(db, User{})
	db.EntityTypes[usersColl.Id] = usersColl
	jobsColl := NewEntityType(db, Job{})
	db.EntityTypes[jobsColl.Id] = jobsColl
	return &MyDb{
		BaseDB: db,
		Users:  usersColl.Store,
		Jobs:   jobsColl.Store,
	}
}

func InstantiateEntityGroup(db *Db, id string, prefix string) any {
	if (id == reflect.TypeOf(User{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[User]), db)
	} else if (id == reflect.TypeOf(Job{}).Name()) {
		return NewEntityGroup(prefix, db.EntityTypes[id].(*EntityType[Job]), db)
	}
	return nil
}

var logger = &Logger{}

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

//export run
func run(a int64) int64 {

	logger.Log("hello keyhan !")

	output(bytesToPointer([]byte("{ \"hello\": \"world\" }")))
	return 0
}

func main() {
	fmt.Println()
	fmt.Println("module starting...")
	fmt.Println()
}
