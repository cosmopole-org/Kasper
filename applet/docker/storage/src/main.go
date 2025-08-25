package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

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

type SessionInfo struct {
	SessionURI string
	Uploaded   int64
	TotalSize  *int64 // optional
	FileID     string
}

var (
	sessions   = make(map[string]*SessionInfo)
	sessionsMu sync.Mutex
)

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
	registerRoute("/api/createStorage", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("userId")
		b, err := os.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}
		config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
		config.RedirectURL = r.Header.Get("redirectUrl")
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
			res, _ := json.Marshal(map[string]any{"success": true})
			w.Write(res)
		} else if authUrl, ok := client.(string); ok {
			res, _ := json.Marshal(map[string]any{"success": true, "authUrl": authUrl})
			configs[userId] = config
			w.Write(res)
		}
	})
	registerRoute("/api/authorizeStorage", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("userId")
		authCode := r.Header.Get("authCode")
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
		res, _ := json.Marshal(map[string]any{"success": true})
		w.Write(res)
	})
	registerRoute("/api/listFiles", func(w http.ResponseWriter, req *http.Request) {
		userId := req.Header.Get("userId")
		srv := services[userId]
		if srv == nil {
			res, _ := json.Marshal(map[string]any{"success": false})
			w.Write(res)
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
		res, _ := json.Marshal(map[string]any{"success": true, "list": list})
		w.Write(res)
	})
	registerRoute("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		file := r.Header.Get("file")
		userId := r.Header.Get("userId")

		payloadB64, _ := os.ReadFile("/app/input/" + file)
		payload, _ := base64.StdEncoding.DecodeString(string(payloadB64))

		srv := services[userId]
		res, err := srv.Files.Create(&drive.File{
			Name: file,
		}).Media(bytes.NewBuffer(payload), googleapi.ChunkSize(len(payload))).Do()

		if err != nil {
			http.Error(w, "Failed to upload "+err.Error(), http.StatusInternalServerError)
		}

		response, _ := json.Marshal(map[string]any{"success": true, "fileId": res.Id})
		w.Write(response)
	})
	registerRoute("/api/download", func(w http.ResponseWriter, req *http.Request) {
		userId := req.Header.Get("userId")
		fileId := req.Header.Get("fileId")
		srv := services[userId]
		res, err := srv.Files.Get(fileId).Download()
		if err != nil {
			log.Fatalf("Could not download file: %v", err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Failed to read downloaded data: %v", err)
		}
		resp, _ := json.Marshal(map[string]any{"success": true, "data": base64.StdEncoding.EncodeToString(data)})
		w.Write(resp)
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
	var fileId string
	flag.StringVar(&fileId, "fileId", "", "")
	var fileCT string
	flag.StringVar(&fileCT, "fileContentType", "", "")
	var redirectUrl string
	flag.StringVar(&redirectUrl, "redirectUrl", "", "")
	var totalSize int
	flag.IntVar(&totalSize, "totalSize", 0, "")
	flag.Parse()

	if command == "adminInit" {
		runHttpServer()
	} else if command == "createStorage" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/createStorage", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("userId", userId)
		req.Header.Set("redirectUrl", redirectUrl)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	} else if command == "authorizeStorage" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/authorizeStorage", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("userId", userId)
		req.Header.Set("authCode", authCode)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	} else if command == "listFiles" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/listFiles", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("userId", userId)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	} else if command == "upload" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/upload", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("userId", userId)
		req.Header.Set("file", file)
		req.Header.Set("fileContentType", fileCT)
		req.Header.Set("X-Total-Size", fmt.Sprintf("%d", totalSize))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	} else if command == "download" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/download", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("userId", userId)
		req.Header.Set("fileId", fileId)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		log.Println(string(body))
	}
}
