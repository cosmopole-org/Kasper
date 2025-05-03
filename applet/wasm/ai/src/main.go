package main

import (
	model "applet/src/models"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

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

func (c *Chain) SubmitAppletPacketTrx(pointId string, targetMachineId string, key string, tag string, input any) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	tagO, tagL := bytesToPointer([]byte("00" + tag))
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key))
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

func (c *Chain) SubmitAppletFileTrx(pointId string, targetMachineId string, fileId string, tag string) []byte {
	tmO, tmL := bytesToPointer([]byte(targetMachineId))
	tagO, tagL := bytesToPointer([]byte("10" + tag))
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + "/storage/upload"))
	inputO, inputL := bytesToPointer([]byte(fileId))
	resP := submitOnchainTrx(tmO, tmL, keyO, keyL, inputO, inputL, tagO, tagL)
	result := pointerToBytes(resP)
	return result
}

func (c *Chain) SubmitBasePacketTrx(pointId string, key string, tag string, input []byte) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + key))
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

func (c *Chain) SubmitBaseFileTrx(pointId string, fileId string, tag string) []byte {
	keyO, keyL := bytesToPointer([]byte(pointId + "|" + "/storage/upload"))
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

func SendSignal(typ string, pointId string, userId string, data string) {
	kp, kl := bytesToPointer([]byte(typ))
	cp, cl := bytesToPointer([]byte(pointId))
	cop, col := bytesToPointer([]byte(userId))
	cop2, col2 := bytesToPointer([]byte(data))
	signalPoint(kp, kl, cp, cl, cop, col, cop2, col2)
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

//export run
func run(a int64) int64 {

	input := map[string]any{}
	signal := ParseArgs(a)
	err := json.Unmarshal([]byte(signal.Data), &input)
	if err != nil {
		logger.Log(err.Error())
	}

	attachmentStr := input["attachment"].(string)
	attachment := map[string]any{}
	e := json.Unmarshal([]byte(attachmentStr), &attachment)
	if e != nil {
		logger.Log("parsing attachment failed")
	}
	action := attachment["action"].(string)

	if action == "init" {
		vm := Vm{}
		filesMap := map[string]string{}
		srcFiles := attachment["initSrcFiles"].(map[string]any)
		for k, v := range srcFiles {
			filesMap[k] = v.(string)
		}
		modelFile := vm.RunDocker("ai_init", "convnet_1_"+strings.Join(strings.Split(signal.Point.Id, "@"), "_"), filesMap)
		chain := Chain{}
		attachmentNext := map[string]any{
			"srcFiles":         attachment["mergeSrcFiles"].(map[string]any),
			"srcFilesNext":     attachment["trainSrcFiles"].(map[string]any),
			"srcFilesNextNext": attachment["aggSrcFiles"].(map[string]any),
			"trainPointId":     signal.Point.Id,
			"action":           "merge",
		}
		chain.PlantTrigger(2, attachment["aggPointId"].(string), "convnet1", attachmentNext)
		chain.SubmitBaseFileTrx(attachment["aggPointId"].(string), modelFile.Id, "convnet1")
		output(bytesToPointer([]byte("{ \"response\": \"initialized the model\" }")))
		return 0
	} else if action == "merge" {
		vm := Vm{}
		filesMap := map[string]string{}
		srcFiles := attachment["srcFiles"].(map[string]any)
		for k, v := range srcFiles {
			filesMap[k] = v.(string)
		}

		getPayload := func(index int) {
			payload := map[string]any{}
			logger.Log(input["payloads"].([]any)[index].(string))
			e = json.Unmarshal([]byte(input["payloads"].([]any)[index].(string)), &payload)
			if e != nil {
				logger.Log(e.Error())
				return
			}
			modelFileId := payload["file"].(map[string]any)["id"].(string)
			filesMap[modelFileId] = fmt.Sprintf("model%d", (index + 1))
		}
		getPayload(0)
		getPayload(1)

		modelFile := vm.RunDocker("ai_merge", "mergenet_"+strings.Join(strings.Split(attachment["trainPointId"].(string), "@"), "_"), filesMap)
		chain := Chain{}
		attachmentNext := map[string]any{
			"srcFiles":     attachment["srcFilesNext"].(map[string]any),
			"srcFilesNext": attachment["srcFilesNextNext"].(map[string]any),
			"aggPointId":   signal.Point.Id,
			"action":       "train",
		}
		chain.PlantTrigger(1, attachment["trainPointId"].(string), "convnet_"+attachment["trainPointId"].(string), attachmentNext)
		chain.SubmitBaseFileTrx(attachment["trainPointId"].(string), modelFile.Id, "convnet_"+attachment["trainPointId"].(string))
		output(bytesToPointer([]byte("{ \"response\": \"merged the models\" }")))
		return 0
	} else if action == "train" {
		vm := Vm{}
		filesMap := map[string]string{}
		srcFiles := attachment["srcFiles"].(map[string]any)
		for k, v := range srcFiles {
			filesMap[k] = v.(string)
		}

		payload := map[string]any{}
		e = json.Unmarshal([]byte(input["payloads"].([]any)[0].(string)), &payload)
		if e != nil {
			logger.Log(e.Error())
			return 0
		}
		modelFileId := payload["file"].(map[string]any)["id"].(string)
		filesMap[modelFileId] = "model"

		modelFile := vm.RunDocker("ai_train", "convnet_"+strings.Join(strings.Split(signal.Point.Id, "@"), "_"), filesMap)
		chain := Chain{}
		attachmentNext := map[string]any{
			"srcFiles":     attachment["srcFilesNext"].(map[string]any),
			"trainPointId": signal.Point.Id,
			"action":       "aggregate",
		}
		chain.PlantTrigger(2, attachment["aggPointId"].(string), "convnet2", attachmentNext)
		chain.SubmitBaseFileTrx(attachment["aggPointId"].(string), modelFile.Id, "convnet2")
		output(bytesToPointer([]byte("{ \"response\": \"trained the model\" }")))
		return 0
	} else if action == "aggregate" {
		vm := Vm{}
		filesMap := map[string]string{}
		for k, v := range attachment["srcFiles"].(map[string]any) {
			filesMap[k] = v.(string)
		}

		getPayload := func(index int) {
			payload := map[string]any{}
			e = json.Unmarshal([]byte(input["payloads"].([]any)[index].(string)), &payload)
			if e != nil {
				logger.Log(e.Error())
				return
			}
			modelFileId := payload["file"].(map[string]any)["id"].(string)
			filesMap[modelFileId] = fmt.Sprintf("model%d", (index + 1))
		}
		getPayload(0)
		getPayload(1)

		modelFile := vm.RunDocker("ai_agg", "aggnet_"+strings.Join(strings.Split(attachment["trainPointId"].(string), "@"), "_"), filesMap)
		chain := Chain{}
		chain.SubmitBaseFileTrx(signal.Point.Id, modelFile.Id, "")
		output(bytesToPointer([]byte("{ \"response\": \"aggregated the models\" }")))
		return 0
	}
	return 0
}

func main() {
	fmt.Println()
	fmt.Println("module starting...")
	fmt.Println()
}
