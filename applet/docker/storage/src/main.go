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
	"strconv"
	"strings"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"

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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Println("Unable to cache oauth token: " + err.Error())
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
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
				log.Fatalf("Unable to retrieve Drive client: %v", err)
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
		fileId := r.Header.Get("fileId")
		userId := r.Header.Get("userId")

		fileKey := userId + "_" + fileId

		token, ok := tokens[userId]
		if !ok {
			http.Error(w, "Failed to find user token", http.StatusInternalServerError)
			return
		}

		payloadB64, _ := os.ReadFile("/app/input/" + file)
		payload, _ := base64.StdEncoding.DecodeString(string(payloadB64))

		totalHeader := r.Header.Get("X-Total-Size")
		var total *int64
		if totalHeader != "" {
			t, _ := strconv.ParseInt(totalHeader, 10, 64)
			total = &t
		}

		sessionsMu.Lock()
		sess := sessions[fileKey]
		sessionsMu.Unlock()

		if sess == nil {
			endpoint := "https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable"
			method := "POST"
			if fileId != "" {
				endpoint = fmt.Sprintf("https://www.googleapis.com/upload/drive/v3/files/%s?uploadType=resumable", fileId)
				method = "PATCH"
			}

			metaBytes, _ := json.Marshal(map[string]any{"name": file})
			meta := string(metaBytes)

			initReq, _ := http.NewRequest(method, endpoint, io.NopCloser(strings.NewReader(meta)))
			initReq.Header.Set("Authorization", "Bearer "+token)
			initReq.Header.Set("Content-Type", "application/json; charset=UTF-8")
			initReq.Header.Set("X-Upload-Content-Type", r.Header.Get("fileContentType"))
			if total != nil {
				initReq.Header.Set("X-Upload-Content-Length", fmt.Sprint(*total))
			}

			initResp, err := http.DefaultClient.Do(initReq)
			if err != nil {
				http.Error(w, "Failed to init session: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer initResp.Body.Close()

			sessionURI := initResp.Header.Get("Location")
			if sessionURI == "" {
				http.Error(w, "No resumable session URI", http.StatusInternalServerError)
				return
			}

			sess = &SessionInfo{SessionURI: sessionURI, Uploaded: 0, TotalSize: total, FileID: fileId}
			sessionsMu.Lock()
			sessions[fileKey] = sess
			sessionsMu.Unlock()
		}

		// Step 2: Upload chunk via PUT to session URI
		chunkSize := int64(len(payload))
		end := sess.Uploaded + chunkSize - 1
		totalStr := "*"
		if sess.TotalSize != nil {
			totalStr = fmt.Sprint(*sess.TotalSize)
		}

		ctx := context.Background()

		putReq, _ := http.NewRequest("PUT", sess.SessionURI, bytes.NewBuffer(payload))
		putReq.Header.Set("Content-Length", fmt.Sprint(chunkSize))
		putReq.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%s", sess.Uploaded, end, totalStr))
		putReq = putReq.WithContext(ctx)
		putReq.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(putReq)
		if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusPermanentRedirect) {
			status := "<no response>"
			if resp != nil {
				status = resp.Status
				resp.Body.Close()
			}
			http.Error(w, "Upload failed: "+status, http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Retrieve file metadata after upload completion
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, resp.Body); err != nil {
			http.Error(w, "Failed to read response body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		result := buf.Bytes()
		log.Println(string(result))

		sess.Uploaded += chunkSize
		fmt.Fprintf(w, "Uploaded total: %d bytes\n", sess.Uploaded)
		os.Remove("/app/input/" + file)

		// Parse the response to extract file ID
		var metadata map[string]interface{}
		if err := json.Unmarshal(result, &metadata); err != nil {
			response, _ := json.Marshal(map[string]any{"success": true})
			w.Write(response)
			return
		} else {
			if id, ok := metadata["id"].(string); ok {
				sess.FileID = id
			} else {
				http.Error(w, "File ID not found in response", http.StatusInternalServerError)
				return
			}
			if sess.FileID != "" {
				delete(sessions, fileKey)
				response, _ := json.Marshal(map[string]any{"success": true, "fileId": sess.FileID})
				w.Write(response)
			} else {
				response, _ := json.Marshal(map[string]any{"success": true, "key": file})
				w.Write(response)
			}
		}
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
