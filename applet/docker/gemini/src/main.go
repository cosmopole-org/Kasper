package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/genai"
)

type Chat struct {
	Key     string
	History []*genai.Content
}

var chats = map[string]*Chat{}

func registerRoute(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		handler(w, r)
	})
}

var callbacks = map[string]func(map[string]any) []byte{}

func runHttpServer() {
	registerRoute("/api/interact", func(w http.ResponseWriter, r *http.Request) {
		pointId := r.Header.Get("pointId")
		message := r.Header.Get("message")

		message, _ = url.QueryUnescape(message)
		inp := map[string]any{}
		json.Unmarshal([]byte(message), &inp)

		message = inp["text"].(string)
		machinesMeta := inp["machines"].([]any)

		tools := []*genai.FunctionDeclaration{}

		for _, metaRaw := range machinesMeta {
			metaObj := metaRaw.(map[string]any)
			tools := metaObj["tools"].([]any)
			for _, toolRaw := range tools {
				toolObj := toolRaw.(map[string]any)
				params := map[string]*genai.Schema{}
				for k, v := range toolObj["args"].(map[string]any) {
					params[k] = &genai.Schema{
						Title:       k,
						Type:        v.(map[string]any)["type"].(genai.Type),
						Description: v.(map[string]any)["desc"].(string),
					}
				}
				tools = append(tools, &genai.FunctionDeclaration{
					Name: toolObj["name"].(string),
					Parameters: &genai.Schema{
						Type:       genai.TypeObject,
						Properties: params,
					},
					Description: "save a key value data into redis",
				})
			}
		}

		temp := strings.ReplaceAll(message, "\n", "")
		temp = strings.ReplaceAll(temp, "\t", "")
		temp = strings.Trim(temp, " ")
		temp = temp[1 : len(temp)-1]

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
			output, _ := json.Marshal(map[string]any{"type": "text-message", "text": "context reset"})
			w.Write(output)
			return
		}

		ctx := context.Background()
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  "AIzaSyAekCwMAh1HlKtogiUVsfkMEEzOcN1pRSs",
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			log.Fatal(err)
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

		chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{
					FunctionDeclarations: tools,
				},
			},
		}, history)

		res, _ := chat.SendMessage(ctx, genai.Part{Text: message})

		fc := res.FunctionCalls()
		if len(fc) > 0 {
			toolName := fc[0].Name
			args := fc[0].Args
			id := fc[0].ID
			callbacks[pointId] = func(result map[string]any) []byte {
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
							FunctionDeclarations: []*genai.FunctionDeclaration{
								{
									Name: "set",
									Parameters: &genai.Schema{
										Type: genai.TypeObject,
										Properties: map[string]*genai.Schema{
											"key": {
												Title:       "key",
												Type:        genai.TypeString,
												Description: "key of the pair to be saved to redis",
											},
											"value": {
												Title:       "value",
												Type:        genai.TypeString,
												Description: "value of the pair to be saved to redis",
											},
										},
									},
									Description: "save a key value data into redis",
								},
								{
									Name: "get",
									Parameters: &genai.Schema{
										Type: genai.TypeObject,
										Properties: map[string]*genai.Schema{
											"key": {
												Title:       "key",
												Type:        genai.TypeString,
												Description: "key of the pair to be fetched by from redis",
											},
										},
									},
									Description: "get a key value data from redis",
								},
							},
						},
					},
				}, history)
				res, _ = chat.SendMessage(ctx, genai.Part{FunctionResponse: &genai.FunctionResponse{ID: id, Name: toolName, Response: result}})
				response := res.Candidates[0].Content.Parts[0].Text
				chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
				output, _ := json.Marshal(map[string]any{"type": "text-message", "text": response})
				return output
			}
			output, _ := json.Marshal(map[string]any{"type": "tool-call", "data": map[string]any{"name": toolName, "args": args, "type": "execute"}})
			w.Write(output)
		} else if len(res.Candidates) > 0 {
			response := res.Candidates[0].Content.Parts[0].Text
			chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
			output, _ := json.Marshal(map[string]any{"type": "text-message", "text": response})
			w.Write(output)
		}
	})
	registerRoute("/api/interactCallback", func(w http.ResponseWriter, r *http.Request) {
		pointId := r.Header.Get("pointId")
		message := r.Header.Get("message")

		message, _ = url.QueryUnescape(message)
		message = message[1 : len(message)-1]

		log.Println(message)

		params := map[string]any{}
		json.Unmarshal([]byte(message), &params)

		log.Println(params)

		cb := callbacks[pointId]
		if cb != nil {
			output := cb(params)
			delete(callbacks, pointId)
			w.Write(output)
		}
	})
	registerRoute("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		message := r.Header.Get("message")

		message, _ = url.QueryUnescape(message)

		ctx := context.Background()
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  "AIzaSyAekCwMAh1HlKtogiUVsfkMEEzOcN1pRSs",
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			log.Fatal(err)
		}

		res, _ := client.Models.GenerateContent(ctx, "gemini-2.5-flash", genai.Text(message), nil)

		if len(res.Candidates) > 0 {
			response := res.Candidates[0].Content.Parts[0].Text
			w.Write([]byte(response))
		}
	})
	http.ListenAndServe(":3000", nil)
}

func main() {
	var command string
	flag.StringVar(&command, "command", "", "")
	var authCode string
	flag.StringVar(&authCode, "authCode", "", "")
	var userId string
	flag.StringVar(&userId, "userId", "", "")
	var file string
	flag.StringVar(&file, "file", "", "")
	var pointId string
	flag.StringVar(&pointId, "pointId", "", "")
	var fileCT string
	flag.StringVar(&fileCT, "fileContentType", "", "")
	var message string
	flag.StringVar(&message, "message", "", "")
	flag.Parse()

	if command == "adminInit" {
		runHttpServer()
	} else if command == "interact" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/interact", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("pointId", pointId)
		req.Header.Set("message", message)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	} else if command == "interactCallback" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/interactCallback", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("pointId", pointId)
		req.Header.Set("message", message)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	} else if command == "generate" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/generate", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("message", message)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	}
}
