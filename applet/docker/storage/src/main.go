package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	"google.golang.org/api/option"
)

var callbacks = map[int64]func([]byte){}

func getClient(config *oauth2.Config) any {
	return getTokenFromWeb(config)
}

func getTokenFromWeb(config *oauth2.Config) string {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)
	return authURL
}

func authorizeAndGEtToken(userId string, authCode string) *http.Client {
	config := configs[userId]
	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Println("Unable to retrieve token from web " + err.Error())
	}
	tokens[userId] = tok.AccessToken
	return config.Client(context.Background(), tok)
}

var tokens = map[string]string{}
var clients = map[string]*http.Client{}
var configs = map[string]*oauth2.Config{}
var services = map[string]*drive.Service{}

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
	Id         string
	IsPublic   bool
	LastUpdate int64
}

func (d *Point) Push() {
	isPubByte := byte(0x02)
	if d.IsPublic == true {
		isPubByte = 0x01
	}
	luBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(luBytes, uint64(d.LastUpdate))
	obj := map[string]string{
		"id":         base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"isPublic":   base64.StdEncoding.EncodeToString([]byte{isPubByte}),
		"lastUpdate": base64.StdEncoding.EncodeToString(luBytes),
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
		if m["isPublic"][0] == byte(0x01) {
			d.IsPublic = true
		} else {
			d.IsPublic = false
		}
		d.LastUpdate = int64(binary.LittleEndian.Uint64(m["lastUpdate"]))
		return true
	}
	return false
}

func (d *Point) Parse(m map[string][]byte) {
	if m["isPublic"][0] == byte(0x01) {
		d.IsPublic = true
	} else {
		d.IsPublic = false
	}
	d.LastUpdate = int64(binary.LittleEndian.Uint64(m["lastUpdate"]))
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

func (d *Doc) Push() {
	obj := map[string]string{
		"id":        base64.StdEncoding.EncodeToString([]byte(d.Id)),
		"title":     base64.StdEncoding.EncodeToString([]byte(d.Title)),
		"fileId":    base64.StdEncoding.EncodeToString([]byte(d.FileId)),
		"mimeType":  base64.StdEncoding.EncodeToString([]byte(d.MimeType)),
		"category":  base64.StdEncoding.EncodeToString([]byte(d.Category)),
		"creatorId": base64.StdEncoding.EncodeToString([]byte(d.CreatorId)),
		"pointId":   base64.StdEncoding.EncodeToString([]byte(d.PointId)),
	}
	c := make(chan int, 1)
	dbPutObj("Doc", d.Id, obj, func() {
		c <- 1
	})
	<-c
}

func (d *Doc) Pull() bool {
	c := make(chan map[string][]byte, 1)
	dbGetObj("Point", d.Id, func(m map[string][]byte) {
		c <- m
	})
	m := <-c
	if len(m) > 0 {
		d.Category = string(m["category"])
		d.CreatorId = string(m["creatorId"])
		d.FileId = string(m["fileId"])
		d.MimeType = string(m["mimeType"])
		d.PointId = string(m["pointId"])
		d.Title = string(m["title"])
		return true
	} else {
		return false
	}
}

func (d *Doc) Parse(m map[string][]byte) {
	d.Category = string(m["category"])
	d.CreatorId = string(m["creatorId"])
	d.FileId = string(m["fileId"])
	d.MimeType = string(m["mimeType"])
	d.PointId = string(m["pointId"])
	d.Title = string(m["title"])
}

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
		if input["type"] == "createStorage" {
			redirectUrl := input["redirectUrl"].(string)
			b, err := os.ReadFile("credentials.json")
			if err != nil {
				log.Fatalf("Unable to read client secret file: %v", err)
			}
			config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
			config.RedirectURL = redirectUrl
			config.Scopes = append(config.Scopes, "https://www.googleapis.com/auth/drive.file")
			if err != nil {
				log.Fatalf("Unable to parse client secret file to config: %v", err)
			}
			client := getClient(config)
			if httpClient, ok := client.(*http.Client); ok {
				clients[userId] = httpClient
				ctx := context.Background()
				srv, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
				if err != nil {
					log.Fatalf("Unable to retrieve Drive client: %v", err)
				}
				services[userId] = srv
				configs[userId] = config
				signalPoint("single", pointId, userId, map[string]any{"type": "initStorageRes", "response": map[string]any{"success": true}})
			} else if authUrl, ok := client.(string); ok {
				configs[userId] = config
				signalPoint("single", pointId, userId, map[string]any{"type": "initStorageRes", "response": map[string]any{"success": true, "authUrl": authUrl}})
			}
		} else if input["type"] == "authorizeStorage" {
			authCode := input["authCode"].(string)
			client := authorizeAndGEtToken(userId, authCode)
			if client != nil {
				clients[userId] = client
				ctx := context.Background()
				srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
				if err != nil {
					log.Println("Unable to retrieve Drive client: " + err.Error())
				}
				services[userId] = srv
			}
			user := User{Id: userId, Name: "-", AuthCode: authCode}
			user.Push()
			signalPoint("single", pointId, userId, map[string]any{"type": "authStorageRes", "response": map[string]any{"success": true}})
		} else if input["type"] == "listFiles" {
			srv := services[userId]
			if srv == nil {
				signalPoint("single", pointId, userId, map[string]any{"type": "listFilesRes", "response": map[string]any{"success": false}})
				return
			}
			r, err := srv.Files.List().PageSize(10).
				Fields("nextPageToken, files(id, name)").Do()
			if err != nil {
				log.Fatalf("Unable to retrieve files: %v", err)
			}
			fmt.Println("Files:")
			list := []map[string]any{}
			if len(r.Files) == 0 {
				fmt.Println("No files found.")
			} else {
				for _, i := range r.Files {
					fmt.Printf("%s (%s)\n", i.Name, i.Id)
					list = append(list, map[string]any{"fileName": i.Name, "fileId": i.Id})
				}
			}
			signalPoint("single", pointId, userId, map[string]any{"type": "listFilesRes", "response": map[string]any{"success": true, "list": list}})
		} else if input["type"] == "pointFiles" {
			docs := []*Doc{}
			point := Point{Id: pointId}
			if point.Pull() {
				c := make(chan int, 1)
				dbGetObjsByPrefix("Doc", "pointDocs::"+pointId+"::", 0, 100, func(m map[string]map[string][]byte) {
					for k, v := range m {
						doc := Doc{Id: k}
						doc.Parse(v)
						docs = append(docs, &doc)
					}
					c <- 1
				})
				<-c
			}
			signalPoint("single", pointId, userId, map[string]any{"type": "pointFilesRes", "response": map[string]any{"success": true, "docs": docs}})
		} else if input["type"] == "listTopMedia" {
			dbGetObjs("Point", 0, 100, func(m map[string]map[string][]byte) {
				pointsList := make([]*Point, len(m))
				for pointId, pointRaw := range m {
					point := Point{Id: pointId}
					point.Parse(pointRaw)
					pointsList = append(pointsList, &point)
				}
				slices.SortFunc(pointsList, func(a *Point, b *Point) int {
					return int(b.LastUpdate - a.LastUpdate)
				})
				docs := []*Doc{}
				pointsList = pointsList[:int(math.Min(float64(len(pointsList)), 5))]
				for _, p := range pointsList {
					c := make(chan int, 1)
					dbGetObjsByPrefix("Doc", "pointDocs::"+p.Id+"::", 0, 100, func(m map[string]map[string][]byte) {
						for k, v := range m {
							doc := Doc{Id: k}
							doc.Parse(v)
							docs = append(docs, &doc)
						}
						c <- 1
					})
					<-c
				}
				signalPoint("single", pointId, userId, map[string]any{"type": "listTopMediaRes", "response": map[string]any{"success": true, "docs": docs}})
			})
		} else if input["type"] == "upload" {
			fileName := input["fileName"].(string)
			mimeType := input["mimeType"].(string)
			content, _ := base64.StdEncoding.DecodeString(input["content"].(string))
			srv := services[userId]
			log.Println(len(content))
			res, err := srv.Files.Create(&drive.File{
				Name:     fileName,
				MimeType: mimeType,
			}).Media(bytes.NewReader(content), googleapi.ChunkSize(1024*1024*16)).Do()
			if err != nil {
				signalPoint("single", pointId, userId, map[string]any{"type": "uploadRes", "response": map[string]any{"success": false, "errMsg": err.Error()}})
			}
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
			doc := Doc{Id: uuid.NewString(), Title: fileName, FileId: res.Id, MimeType: mimeType, PointId: pointId, CreatorId: userId, Category: fileType}
			doc.Push()
			point := Point{Id: pointId}
			if !point.Pull() {
				point = Point{Id: pointId, IsPublic: packet["point"].(map[string]any)["isPublic"].(bool), LastUpdate: 0}
				point.Push()
			}
			dbPutLink("pointDocs::"+pointId+"::"+doc.Id, "true")
			point.LastUpdate = time.Now().UnixMilli()
			point.Push()
			signalPoint("single", pointId, userId, map[string]any{"type": "uploadRes", "response": map[string]any{"success": true, "fileId": res.Id}})
		} else if input["type"] == "download" {
			fileId := input["fileId"].(string)
			pointIdInner := input["pointId"].(string)
			doc := Doc{Id: fileId}
			doc.Pull()
			point := Point{Id: pointIdInner}
			point.Pull()
			pId := ""
			if point.IsPublic {
				pId = pointIdInner
			} else if pointId == pointIdInner {
				pId = pointIdInner
			}
			if pId != "" {
				srv := services[doc.CreatorId]
				res, err := srv.Files.Get(doc.FileId).Download()
				if err != nil {
					log.Fatalf("Could not download file: %v", err)
				}
				defer res.Body.Close()
				data, err := io.ReadAll(res.Body)
				if err != nil {
					log.Fatalf("Failed to read downloaded data: %v", err)
				}
				signalPoint("single", pointId, userId, map[string]any{"type": "downloadRes", "response": map[string]any{"success": true, "doc": doc, "data": base64.StdEncoding.EncodeToString(data)}})
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

	log.Println("started storag machine.")

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

func signalPoint(typ string, pointId string, userId string, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	packet, _ := json.Marshal(map[string]any{"key": "signalPoint", "input": map[string]any{
		"type":    typ + "|true",
		"pointId": pointId,
		"userId":  userId,
		"data":    string(dataBytes),
	}})
	writePacket(packet, nil)
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
		"op":     "getObjsByPrefix",
		"objType": objType,
		"prefix": prefix,
		"offset": offset,
		"count":  count,
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
