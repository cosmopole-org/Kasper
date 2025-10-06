package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	model "quickstart/models"
	"strings"
	"sync"

	bleve "github.com/blevesearch/bleve/v2"
)

type User struct {
	Id       string
	Name     string
	AuthCode string
}

func (d *User) Push() {
	obj := map[string]string{
		"id":       base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"name":     base64.StdEncoding.EncodeToString([]byte(d.Name)),
		"authCode": base64.StdEncoding.EncodeToString([]byte(d.AuthCode)),
	}
	c := make(chan int, 1)
	dbPutObj("User", d.Id, obj, func() {
		c <- 1
	})
	<-c
}

func (d *User) Pull() bool {
	c := make(chan map[string][]byte, 1)
	dbGetObj("User", d.Id, func(m map[string][]byte) {
		c <- m
	})
	m := <-c
	if len(m) > 0 {
		d.Name = string(m["name"])
		d.AuthCode = string(m["authCode"])
		return true
	} else {
		return false
	}
}

type Point struct {
	Id            string
	PendingUserId string
}

func (d *Point) Push() {
	obj := map[string]string{
		"id":            base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"pendingUserId": base64.StdEncoding.EncodeToString([]byte(d.PendingUserId)),
	}
	c := make(chan int, 1)
	dbPutObj("Point", d.Id, obj, func() {
		c <- 1
	})
	<-c
}

func (d *Point) Pull() bool {
	c := make(chan map[string][]byte, 1)
	dbGetObj("Point", d.Id, func(m map[string][]byte) {
		c <- m
	})
	m := <-c
	if len(m) > 0 {
		d.PendingUserId = string(m["pendingUserId"])
		return true
	}
	return false
}

func (d *Point) Parse(m map[string][]byte) {
	d.PendingUserId = string(m["pendingUserId"])
}

var callbacks = map[int64]func([]byte){}
var searchers = map[string]bleve.Index{}
var searchersLock sync.Mutex

func processPacket(callbackId int64, data []byte) {
	defer func() {
		r := recover()
		if r != nil {
			var err error
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = errors.New("Unknown error")
			}
			log.Println(err)
		}
	}()
	if callbackId == 0 {
		if string(data) == "{}" {
			return
		}
		packet := map[string]any{}
		err := json.Unmarshal(data, &packet)
		if err != nil {
			log.Println(err)
			return
		}
		data := packet["data"].(string)
		input := map[string]any{}
		err = json.Unmarshal([]byte(data), &input)
		if err != nil {
			log.Println(err)
			return
		}
		userId := packet["user"].(map[string]any)["id"].(string)
		pointId := packet["point"].(map[string]any)["id"].(string)

		if input["type"] == "textMessage" {
			text := strings.Trim(input["text"].(string), " ")
			if strings.HasPrefix(text, "@search") && strings.Contains(text, "/activate") {
				searcherDbPath := "/app/searcher_" + pointId + ".bleve"
				if _, err := os.Stat(searcherDbPath); errors.Is(err, os.ErrNotExist) {
					mapping := bleve.NewIndexMapping()
					searcher, err := bleve.New(searcherDbPath, mapping)
					if err != nil {
						panic(err)
					}
					searchersLock.Lock()
					defer searchersLock.Unlock()
					searchers[pointId] = searcher
				} else {
					searcher, err := bleve.Open(searcherDbPath)
					if err != nil {
						panic(err)
					}
					searchersLock.Lock()
					defer searchersLock.Unlock()
					searchers[pointId] = searcher
				}
			} else if strings.HasPrefix(text, "@search") && strings.Contains(text, "/activateUsername") {
				objName := "username_" + userId
				searcherDbPath := "/app/searcher_" + objName + ".bleve"
				if _, err := os.Stat(searcherDbPath); errors.Is(err, os.ErrNotExist) {
					mapping := bleve.NewIndexMapping()
					searcher, err := bleve.New(searcherDbPath, mapping)
					if err != nil {
						panic(err)
					}
					searchersLock.Lock()
					defer searchersLock.Unlock()
					searchers[objName] = searcher
				} else {
					searcher, err := bleve.Open(searcherDbPath)
					if err != nil {
						panic(err)
					}
					searchersLock.Lock()
					defer searchersLock.Unlock()
					searchers[objName] = searcher
				}
			} else {
				searchersLock.Lock()
				defer searchersLock.Unlock()
				searcher, ok := searchers[pointId]
				if ok {
					searcher.Index(packet["id"].(string), packet)
				}
				parts := strings.Split(input["text"].(string), " ")
				for _, word := range parts {
					if strings.HasPrefix(word, "@") {
						ch := make(chan []byte)
						submitOffchainBaseTrx(pointId, "/users/getByUsername", "", "", "", "", map[string]any{
							"username": word[1:],
						}, func(b []byte) {
							ch <- b
						})
						result := <-ch
						res := model.GetOutput{}
						json.Unmarshal(result, &res)
						uname := res.User["username"]
						if uname == word[1:] {
							searcher, ok := searchers["username_"+res.User["id"].(string)]
							if ok {
								searcher.Index(pointId+"|"+packet["id"].(string), res.User["username"].(string))
							}
						}
					}
				}

			}
		} else if input["type"] == "search" {
			quest := packet["user"].(map[string]any)["username"].(string)
			requestId := input["requestId"].(string)
			query := bleve.NewMatchQuery(quest)
			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Explain = true
			searchRequest.Fields = []string{"data"}

			searchersLock.Lock()
			defer searchersLock.Unlock()
			searcher, ok := searchers["username_"+userId]

			if ok {

				searchResult, err := searcher.Search(searchRequest)
				if err != nil {
					log.Println(err)
					signalPoint("single", pointId, userId, map[string]any{"type": "searchRes", "ids": []string{}, "callbackId": requestId}, true)
					return
				}
				ids := []string{}
				for _, hit := range searchResult.Hits {
					ids = append(ids, hit.ID)
				}
				signalPoint("single", pointId, userId, map[string]any{"type": "searchRes", "ids": ids, "callbackId": requestId}, true)
			} else {
				signalPoint("single", pointId, userId, map[string]any{"type": "searchRes", "ids": []string{}, "callbackId": requestId}, true)
			}
		} else if input["type"] == "searchMentions" {
			quest := input["query"].(string)
			requestId := input["requestId"].(string)
			query := bleve.NewMatchQuery(quest)
			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Explain = true
			searchRequest.Fields = []string{"data"}

			searchersLock.Lock()
			defer searchersLock.Unlock()
			searcher, ok := searchers[pointId]

			if ok {

				searchResult, err := searcher.Search(searchRequest)
				if err != nil {
					log.Println(err)
					signalPoint("single", pointId, userId, map[string]any{"type": "searchRes", "ids": []string{}, "callbackId": requestId}, true)
					return
				}
				ids := []string{}
				for _, hit := range searchResult.Hits {
					ids = append(ids, hit.ID)
				}
				signalPoint("single", pointId, userId, map[string]any{"type": "searchRes", "ids": ids, "callbackId": requestId}, true)
			} else {
				signalPoint("single", pointId, userId, map[string]any{"type": "searchRes", "ids": []string{}, "callbackId": requestId}, true)
			}
		}
	} else {
		cb := callbacks[callbackId]
		cb(data)
		delete(callbacks, callbackId)
	}
}

var lock sync.Mutex
var conn net.Conn

func main() {

	log.Println("started searcher machine.")

	var err error
	conn, err = net.Dial("tcp", "10.10.0.3:8084")
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()
	log.Println("Container client connected")

	r := bufio.NewReader(conn)
	for {
		var ln uint32
		if err := binary.Read(r, binary.LittleEndian, &ln); err != nil {
			if err != io.EOF {
				log.Printf("read len err: %v", err)
			}
			os.Exit(0)
		}
		var callbackId uint64
		if err := binary.Read(r, binary.LittleEndian, &callbackId); err != nil {
			if err != io.EOF {
				log.Printf("read len err: %v", err)
			}
			os.Exit(0)
		}
		buf := make([]byte, ln)
		if _, err := io.ReadFull(r, buf); err != nil {
			log.Printf("read body err: %v", err)
			os.Exit(0)
		}
		log.Printf("recv: %s", string(buf))
		go func() {
			processPacket(int64(callbackId), buf)
		}()
	}
}

var cbCounter = int64(0)

func writePacket(data []byte, callback func([]byte)) {
	lock.Lock()
	defer lock.Unlock()
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
	conn.Write(lenBytes)
	if callback == nil {
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(0))
		conn.Write(cbId)
	} else {
		cbCounter++
		callbackId := cbCounter
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(cbCounter))
		callbacks[callbackId] = callback
		conn.Write(cbId)
	}
	conn.Write(data)
}

func signalPoint(typ string, pointId string, userId string, data any, temp ...bool) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	if len(temp) > 0 {
		isTemp := "false"
		if temp[0] {
			isTemp = "true"
		}
		packet, _ := json.Marshal(map[string]any{"key": "signalPoint", "input": map[string]any{
			"type":    typ + "|" + isTemp,
			"pointId": pointId,
			"userId":  userId,
			"data":    string(dataBytes),
		}})
		writePacket(packet, nil)
	} else {
		packet, _ := json.Marshal(map[string]any{"key": "signalPoint", "input": map[string]any{
			"type":    typ + "|false",
			"pointId": pointId,
			"userId":  userId,
			"data":    string(dataBytes),
		}})
		writePacket(packet, nil)
	}
}

func submitOffchainBaseTrx(pointId string, key string, requesterUserId string, requesterSignature string, tokenId string, tag string, input any, cb func([]byte)) {
	inp, _ := json.Marshal(input)
	packet, _ := json.Marshal(map[string]any{"key": "submitOnchainTrx", "input": map[string]any{
		"targetMachineId":    "-",
		"isRequesterOnchain": false,
		"key":                pointId + "|" + key + "|" + requesterUserId + "|" + requesterSignature + "|" + tokenId + "|" + "false" + "|",
		"pointId":            pointId,
		"isFile":             false,
		"isBase":             true,
		"tag":                tag,
		"packet":             string(inp),
	}})
	writePacket(packet, func(output []byte) {
		cb(output)
	})
}

func dbPutObj(typ string, objId string, obj map[string]string, cb func()) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "putObj",
		"objType": typ,
		"objId":   objId,
		"obj":     obj,
	}})
	writePacket(packet, func([]byte) {
		cb()
	})
}

func dbGetObj(typ string, objId string, cb func(map[string][]byte)) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "getObj",
		"objType": typ,
		"objId":   objId,
	}})
	writePacket(packet, func(b []byte) {
		result := map[string][]byte{}
		json.Unmarshal(b, &result)
		cb(result)
	})
}

func dbPutLink(linkKey string, linkVal string) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":  "putLink",
		"key": linkKey,
		"val": linkVal,
	}})
	writePacket(packet, nil)
}

func dbGetObjsByPrefix(objType string, prefix string, offset int, count int, cb func(map[string]map[string][]byte)) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "getObjsByPrefix",
		"objType": objType,
		"prefix":  prefix,
		"offset":  offset,
		"count":   count,
	}})
	writePacket(packet, func(b []byte) {
		result := map[string]map[string][]byte{}
		json.Unmarshal(b, &result)
		cb(result)
	})
}

func dbGetObjs(typ string, offset int, count int, cb func(map[string]map[string][]byte)) {
	packet, _ := json.Marshal(map[string]any{"key": "dbOp", "input": map[string]any{
		"op":      "getObjs",
		"objType": typ,
		"offset":  offset,
		"count":   count,
	}})
	writePacket(packet, func(b []byte) {
		result := map[string]map[string][]byte{}
		json.Unmarshal(b, &result)
		cb(result)
	})
}
