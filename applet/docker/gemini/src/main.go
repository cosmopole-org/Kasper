package main

import (
	"bytes"
	"context"
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

func runHttpServer() {
	registerRoute("/api/interact", func(w http.ResponseWriter, r *http.Request) {
		pointId := r.Header.Get("pointId")
		message := r.Header.Get("message")

		message, _ = url.QueryUnescape(message)

		message = strings.ReplaceAll(message, "\n", "")
		message = strings.ReplaceAll(message, "\t", "")
		message = message[1 : len(message)-1]

		log.Println(message)

		if message == "/reset" {
			history := []*genai.Content{}
			chatObj, ok := chats[pointId]
			if ok {
				history = chatObj.History
			} else {
				chatObj = &Chat{Key: pointId, History: history}
				chats[pointId] = chatObj
			}
			chatObj.History = []*genai.Content{}
			w.Write([]byte("context reset"))
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

		chat, _ := client.Chats.Create(ctx, "gemini-2.5-flash", nil, history)
		res, _ := chat.SendMessage(ctx, genai.Part{Text: message})

		if len(res.Candidates) > 0 {
			response := res.Candidates[0].Content.Parts[0].Text
			w.Write([]byte(response))
			chatObj.History = append(chatObj.History, genai.NewContentFromText(response, genai.RoleModel))
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
