package sdk

import (
	model "applet/src/models"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

func Init() {
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
}

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
	ConsoleLog(bytesToPointer([]byte(text)))
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
		Console.Log(typ)
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
	Console.Log("id: " + id)
	return et
}
func (et *EntityType[T]) NewEntity(s *T) *Entity {
	t := reflect.TypeOf(*s)
	r := reflect.ValueOf(s)
	Console.Log("hello 2")
	id := r.Elem().FieldByName("Id").String()
	Console.Log("hello 3")
	props := map[string]any{}
	for i := range t.NumField() {
		name := t.Field(i).Name
		Console.Log("hello 4")
		typ := et.Props[name].Type
		f := r.Elem().FieldByName(name)
		Console.Log("hello 5")
		if strings.Split(typ, "::")[0] == "list" {
			if f.IsValid() {
				if f.CanSet() {
					Console.Log("hello 6")
					if strings.Split(typ, "::")[0] == "list" {
						Console.Log("hello 7 " + typ)
						val := InstantiateEntityGroup(et.Store.Db, strings.Split(typ, "::")[1], et.Id+"::"+id+"::"+name)
						Console.Log("hello 8")
						//props[name] = val
						r.Elem().FieldByName(name).Set(reflect.ValueOf(val))
					}
				}
			}
		}
		c, ok := converters[typ]
		if ok {
			Console.Log("found " + typ + " " + name)
			value := c(f)
			props[name] = value
		} else {
			Console.Log("not found " + typ + " " + name)
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
	Console.Log("hello 12")
	return &EntityGroup[T]{
		Prefix:     prefix,
		EntityType: et,
		Db:         db,
		Map:        map[string]*Entity{},
	}
}

func mapToStruct[T any](db *Db, et *EntityType[T], src map[string]any) T {
	Console.Log("ok 0")
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
						Console.Log("ok 1 " + v.Type + " " + et.Id)
						f.Set(reflect.ValueOf(InstantiateEntityGroup(db, strings.Split(v.Type, "::")[1], et.Id+"::"+id+"::"+k)))
						Console.Log("ok 2")
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
	Console.Log("hello 13")
	b, _ := json.Marshal(e.Props)
	Console.Log("hello 14 " + string(b))
	eg.Db.Put("table::"+eg.EntityType.Id+"::"+e.Id, b)
	Console.Log("hello 15")
	if eg.EntityType.Id != eg.Prefix {
		Console.Log("hello 16 [" + eg.EntityType.Id + "::" + e.Id + "]")
		eg.Db.Put(eg.Prefix+"::"+e.Id, []byte(eg.EntityType.Id+"::"+e.Id))
	}
	Console.Log("hello 17")
	eg.Map[e.Id] = e
}
func (eg *EntityGroup[T]) CreateAndInsert(s *T) {
	Console.Log("hello 1")
	eg.InsertEntity(eg.EntityType.NewEntity(s))
	Console.Log("hello 18")
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
	Console.Log("ok 0.1")
	if eg.Prefix == eg.EntityType.Id {
		Console.Log("ok 0.2")
		b = eg.Db.Get("table::" + eg.Prefix + "::" + id)
	} else {
		Console.Log("ok 0.3")
		b = eg.Db.Get(eg.Prefix + "::" + id)
	}
	Console.Log("ok 0.4")
	if len(b) == 0 {
		Console.Log("ok 0.5")
		return *new(T)
	}
	Console.Log("ok 0.6 " + string(b))
	src := map[string]any{}
	json.Unmarshal(b, &src)
	Console.Log("ok 0.7")
	for k, v := range eg.EntityType.Props {
		Console.Log("ok 0.8 " + v.Type)
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
			Console.Log(reflect.TypeOf(src[k]).Name())
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
			Console.Log(string(b))
		}
	} else {
		Console.Log("hello 16 [" + eg.Prefix + "]")
		keys := eg.Db.GetByPrefix(eg.Prefix)
		for _, keyB := range keys {
			key := string(keyB)
			if eg.Db == nil {
				Console.Log(key + " false")
			} else {
				Console.Log(key + " true")
			}
			b := eg.Db.Get("table::" + key)
			Console.Log(string(b))
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
	Put(kP, kL, vP, vL)
}
func (*Db) Del(key string) {
	kP, kL := bytesToPointer([]byte(key))
	Del(kP, kL)
}
func (*Db) Get(key string) []byte {
	kP, kL := bytesToPointer([]byte(key))
	Console.Log("step 100 " + key)
	val := Get(kP, kL)
	s := strconv.FormatInt(val, 10)
	Console.Log("step 101 " + s)
	return pointerToBytes(val)
}
func (*Db) GetByPrefix(key string) [][]byte {
	kP, kL := bytesToPointer([]byte(key))
	val := GetByPrefix(kP, kL)
	data := pointerToBytes(val)
	Console.Log("first data: " + string(data))
	type BytesInBytes struct {
		Data []string `json:"data"`
	}
	arr := BytesInBytes{Data: []string{}}
	err := json.Unmarshal(data, &arr)
	if err != nil {
		Console.Log(err.Error())
	}
	result := [][]byte{}
	for _, b := range arr.Data {
		Console.Log("first data: " + string(b))
		result = append(result, []byte(b))
	}
	return result
}

type Chain struct {
}

type OffChain struct {
}

func (c *Chain) SubmitAppletPacketTrx(pointId string, targetMachineId string, key string, userId string, signature string, tokenId string, tag string, input any) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	tagO, tagL := bytesToPointer([]byte("00" + tag))
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key + "|" + userId + "|" + signature + "|" + tokenId + "|" + "true"))
	b, e := json.Marshal(input)
	if e != nil {
		Console.Log(e.Error())
		return []byte("{}")
	}
	inputO, inputL := bytesToPointer(b)
	resP := SubmitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitAppletFileTrx(pointId string, targetMachineId string, fileId string, userId string, signature string, tokenId string, tag string) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	tagO, tagL := bytesToPointer([]byte("10" + tag))
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + "/storage/upload" + "|" + userId + "|" + signature + "|" + tokenId))
	inputO, inputL := bytesToPointer([]byte(fileId))
	resP := SubmitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitBasePacketTrx(pointId string, key string, userId string, signature string, tag string, input []byte) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key + "|" + userId + "|" + signature + "|" + "-"))
	tagO, tagL := bytesToPointer([]byte("01" + tag))
	b, e := json.Marshal(input)
	if e != nil {
		Console.Log(e.Error())
		return []byte("{}")
	}
	inputO, inputL := bytesToPointer(b)
	resP := SubmitOnchainTrx(0, 0, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitBaseFileTrx(pointId string, fileId string, userId string, signature string, tag string) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + "/storage/upload" + "|" + userId + "|" + signature + "|" + "-"))
	tagO, tagL := bytesToPointer([]byte("11" + tag))
	inputO, inputL := bytesToPointer([]byte(fileId))
	resP := SubmitOnchainTrx(0, 0, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) PlantTrigger(count int32, pointId string, tag string, input map[string]any) {
	tagO, tagL := bytesToPointer([]byte(tag))
	piO, piL := bytesToPointer([]byte(pointId))
	b, _ := json.Marshal(input)
	inputO, inputL := bytesToPointer(b)
	PlantTrigger(tagO, tagL, inputO, inputL, piO, piL, count)
}

func (c *OffChain) PlantRewoke(seconds int32, pointId string, input map[string]any) {
	tagO, tagL := bytesToPointer([]byte("alarm"))
	piO, piL := bytesToPointer([]byte(pointId))
	b, _ := json.Marshal(input)
	inputO, inputL := bytesToPointer(b)
	PlantTrigger(tagO, tagL, inputO, inputL, piO, piL, seconds)
}

func (c *OffChain) SubmitBaseRequest(pointId string, key string, userId string, signature string, tag string, input any) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key + "|" + userId + "|" + signature + "|" + "-" + "|" + "false"))
	tagO, tagL := bytesToPointer([]byte("10" + tag))
	b, e := json.Marshal(input)
	if e != nil {
		Console.Log(e.Error())
		return []byte("{}")
	}
	inputO, inputL := bytesToPointer(b)
	resP := SubmitOnchainTrx(0, 0, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

type Trx[T any] struct {
	Db       *T
	Chain    *Chain
	Offchain *OffChain
	Signaler *Signaler
	Network  *NetHttp
}

func ParseArgs(a int64) model.Send {
	input := model.Send{}
	str := pointerToBytes(a)
	fmt.Println(string(str))
	e := json.Unmarshal(str, &input)
	if e != nil {
		fmt.Println("unable to parse args as send.")
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
	res := pointerToBytes(RunDocker(kp, kl, vp, vl, cp, cl))
	file := &model.File{}
	err := json.Unmarshal(res, file)
	if err != nil {
		Console.Log(err.Error())
	}
	return file
}

func (vm *Vm) ExecDocker(imageName string, containerName string, command string) string {
	kp, kl := bytesToPointer([]byte(imageName))
	cp, cl := bytesToPointer([]byte(containerName))
	cop, col := bytesToPointer([]byte(command))
	res := pointerToBytes(ExecDocker(kp, kl, cp, cl, cop, col))
	Console.Log(string(res))
	result := map[string]any{}
	json.Unmarshal(res, &result)
	return result["data"].(string)
}

func (vm *Vm) CopyToDocker(imageName string, containerName string, fileName string, content string) {
	kp, kl := bytesToPointer([]byte(imageName))
	cp, cl := bytesToPointer([]byte(containerName))
	cop, col := bytesToPointer([]byte(fileName))
	conp, conl := bytesToPointer([]byte(content))
	CopyToDocker(kp, kl, cp, cl, cop, col, conp, conl)
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
	SignalPoint(kp, kl, cp, cl, cop, col, cop2, col2)
}

// ---------------------------------------------------------------------------

type NetHttp struct{}

func (nh *NetHttp) Request(method string, url string, headers map[string]string, body any) string {
	urlO, urlL := bytesToPointer([]byte(method + "|" + url))
	headersJson, _ := json.Marshal(headers)
	bodyJson, _ := json.Marshal(body)
	headersO, headersL := bytesToPointer(headersJson)
	bodyO, bodyL := bytesToPointer(bodyJson)
	return string(pointerToBytes(HttpPost(urlO, urlL, headersO, headersL, bodyO, bodyL)))
}

var Console = &Logger{}

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
	NewSyncTask(p, l)
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

type Signaler struct {
}

func (s *Signaler) Answer(pointId string, userId string, data any, temp bool) {
	res, _ := json.Marshal(data)
	SendSignal("single", pointId, userId, string(res), temp)
}

func (s *Signaler) Broadcast(pointId string, data any) {
	res, _ := json.Marshal(data)
	SendSignal("broadcast", pointId, "-", string(res), true)
}
