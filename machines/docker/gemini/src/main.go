package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"google.golang.org/genai"
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
	dbGetObj("Doc", d.Id, func(m map[string][]byte) {
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

type Chat struct {
	Key     string
	History []*genai.Content
}

var chats = map[string]*Chat{}

var callbacks = map[int64]func([]byte){}
var toolCallbacks = map[string]func(map[string]any) []byte{}

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
			message := strings.Trim(input["text"].(string), " ")
			if strings.HasPrefix(message, "@gemini") {
				inp := ListPointAppsInput{
					PointId: pointId,
				}
				submitOffchainBaseTrx(pointId, "/points/listApps", "", "", "", "", inp, func(pointsAppsRes []byte) {
					log.Println(string(pointsAppsRes))
					out := ListPointAppsOutput{}
					log.Println("parsing...")
					err := json.Unmarshal(pointsAppsRes, &out)
					if err != nil {
						log.Println(err.Error())
						signalPoint("single", pointId, userId, map[string]any{"type": "textMessage", "text": "an error happended " + string(pointsAppsRes)}, true)
					}
					log.Println("parsed.")

					machinesMeta := []map[string]any{}
					log.Println("starting to extract...")
					for k, v := range out.Machines {
						log.Println(k)
						if v.Identifier == "0" {
							if isMcpRaw, ok := v.Metadata["isMcp"]; ok {
								if isMcp, ok := isMcpRaw.(bool); ok && isMcp {
									log.Println(k + " is mcp")
									machinesMeta = append(machinesMeta, map[string]any{
										"machineId": v.UserId,
										"metadata":  v.Metadata,
									})
								}
							}
						}
					}

					toolsList := []*genai.FunctionDeclaration{}

					toolToMachIdMap := map[string]string{}

					for _, metaRaw := range machinesMeta {
						machineId := metaRaw["machineId"].(string)
						metaObj := metaRaw["metadata"].(map[string]any)
						tools := metaObj["tools"].([]any)
						for _, toolRaw := range tools {
							toolObj := toolRaw.(map[string]any)
							params := map[string]*genai.Schema{}
							for k, v := range toolObj["args"].(map[string]any) {
								t := v.(map[string]any)["type"]
								var typ genai.Type
								if t == "STRING" {
									typ = genai.TypeString
								} else if t == "NUMBER" {
									typ = genai.TypeNumber
								} else if t == "BOOL" {
									typ = genai.TypeBoolean
								} else {
									typ = genai.TypeUnspecified
								}
								params[k] = &genai.Schema{
									Title:       k,
									Type:        typ,
									Description: v.(map[string]any)["desc"].(string),
								}
							}
							toolName := toolObj["name"].(string)
							toolsList = append(toolsList, &genai.FunctionDeclaration{
								Name: toolName,
								Parameters: &genai.Schema{
									Type:       genai.TypeObject,
									Properties: params,
								},
								Description: toolObj["desc"].(string),
							})
							toolToMachIdMap[toolName] = machineId
						}
					}

					temp := strings.ReplaceAll(message, "\n", "")
					temp = strings.ReplaceAll(temp, "\t", "")
					temp = strings.Trim(temp, " ")

					if temp == "/reset" {
						history := []*genai.Content{}
						chatObj, ok := chats[pointId]
						if ok {
							history = chatObj.History
						} else {
							chatObj = &Chat{Key: pointId, History: history}
							chats[pointId] = chatObj
						}
						chatObj.History = []*genai.Content{}
						signalPoint("broadcast", pointId, "-", map[string]any{"type": "textMessage", "text": "context reset"})
						return
					}

					ctx := context.Background()
					client, err := genai.NewClient(ctx, &genai.ClientConfig{
						APIKey:  "AIzaSyAekCwMAh1HlKtogiUVsfkMEEzOcN1pRSs",
						Backend: genai.BackendGeminiAPI,
					})
					if err != nil {
						log.Println(err)
					}

					history := []*genai.Content{}

					chatObj, ok := chats[pointId]
					if ok {
						history = chatObj.History
					} else {
						chatObj = &Chat{Key: pointId, History: history}
						chats[pointId] = chatObj
					}
					chatObj.History = append(chatObj.History, genai.NewContentFromText(message, genai.RoleUser))

					log.Println(toolsList)

					chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
						Tools: []*genai.Tool{
							{
								FunctionDeclarations: toolsList,
							},
						},
					}, history)

					res, _ := chat.SendMessage(ctx, genai.Part{Text: message})

					fc := res.FunctionCalls()
					if len(fc) > 0 {
						toolName := fc[0].Name
						args := fc[0].Args
						id := fc[0].ID
						toolCallbacks[pointId] = func(result map[string]any) []byte {
							history := []*genai.Content{}
							chatObj, ok := chats[pointId]
							if ok {
								history = chatObj.History
							} else {
								chatObj = &Chat{Key: pointId, History: history}
								chats[pointId] = chatObj
							}
							chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
								Tools: []*genai.Tool{
									{
										FunctionDeclarations: toolsList,
									},
								},
							}, history)
							res, _ = chat.SendMessage(ctx, genai.Part{FunctionResponse: &genai.FunctionResponse{ID: id, Name: toolName, Response: result}})
							response := res.Candidates[0].Content.Parts[0].Text
							chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
							return []byte(response)
						}
						point := Point{Id: pointId, PendingUserId: userId}
						point.Push()
						machId := toolToMachIdMap[toolName]
						log.Println(res)
						signalPoint("single", pointId, machId, map[string]any{"name": toolName, "args": args, "type": "execute", "machineId": toolToMachIdMap[toolName]}, true)
					} else if len(res.Candidates) > 0 {
						response := res.Candidates[0].Content.Parts[0].Text
						chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
						signalPoint("broadcast", pointId, "-", map[string]any{"type": "textMessage", "text": response})
					}
				})
			}
		} else if input["type"] == "mcpCallback" {

			params := map[string]any{}
			json.Unmarshal([]byte(input["payload"].(string)), &params)

			log.Println(params)

			cb := toolCallbacks[pointId]
			if cb != nil {
				output := cb(params)
				delete(toolCallbacks, pointId)
				signalPoint("broadcast", pointId, "-", map[string]any{"type": "textMessage", "text": string(output)})
			}
		} else if input["type"] == "generate" {
			ctx := context.Background()
			client, err := genai.NewClient(ctx, &genai.ClientConfig{
				APIKey:  "AIzaSyAekCwMAh1HlKtogiUVsfkMEEzOcN1pRSs",
				Backend: genai.BackendGeminiAPI,
			})
			if err != nil {
				log.Fatal(err)
			}

			res, _ := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(input["text"].(string)), nil)

			if len(res.Candidates) > 0 {
				response := res.Candidates[0].Content.Parts[0].Text
				signalPoint("single", pointId, userId, map[string]any{"type": "textMessage", "text": response})
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
