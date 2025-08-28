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
	"net"
	"net/http"
	"os"
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

type Point struct {
	Id         string
	IsPublic   bool
	LastUpdate int64
	Docs       []*Doc
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

var users = map[string]*User{}
var points = map[string]*Point{}
var docs = map[string]*Doc{}

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
			user := User{Id: userId, Name: "", AuthCode: authCode}
			users[user.Id] = &user
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
			point, ok := points[pointId]
			if ok {
				docs = point.Docs
			}
			signalPoint("single", pointId, userId, map[string]any{"type": "pointFilesRes", "response": map[string]any{"success": true, "docs": docs}})
		} else if input["type"] == "upload" {
			fileName := input["fileName"].(string)
			mimeType := input["mimeType"].(string)
			content, _ := base64.StdEncoding.DecodeString(input["content"].(string))
			srv := services[userId]
			log.Println(len(content))
			res, err := srv.Files.Create(&drive.File{
				Name:     fileName,
				MimeType: mimeType,
			}).Media(bytes.NewReader(content), googleapi.ChunkSize(1024*1024 * 16)).Do()
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
			docs[doc.Id] = &doc
			point, ok := points[pointId]
			if !ok {
				point = &Point{Id: pointId, IsPublic: packet["point"].(map[string]any)["isPublic"].(bool), LastUpdate: 0, Docs: []*Doc{}}
				points[pointId] = point
			}
			point.Docs = append(point.Docs, &doc)
			point.LastUpdate = time.Now().UnixMilli()
			signalPoint("single", pointId, userId, map[string]any{"type": "uploadRes", "response": map[string]any{"success": true, "fileId": res.Id}})
		} else if input["type"] == "download" {
			fileId := input["fileId"].(string)
			pointIdInner := input["pointId"].(string)
			doc := docs[fileId]
			point := points[pointIdInner]
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
		processPacket(int64(callbackId), buf)
	}
}

var cbCounter = int64(0)

func writePacket(data []byte, noCallback bool) {
	lock.Lock()
	defer lock.Unlock()
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
	conn.Write(lenBytes)
	if noCallback {
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(0))
		conn.Write(cbId)
	} else {
		cbCounter++
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(cbCounter))
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
	writePacket(packet, false)
}
