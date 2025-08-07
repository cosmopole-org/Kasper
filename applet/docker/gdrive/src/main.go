package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	// "google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var tokFile = "token.json"

func getClient(config *oauth2.Config) any {
	tok, err := tokenFromFile(tokFile)
	config.RedirectURL = "http://localhost:8080"
	config.Scopes = append(config.Scopes, "https://www.googleapis.com/auth/drive.file")
	if err != nil {
		return getTokenFromWeb(config)
	} else {
		return config.Client(context.Background(), tok)
	}
}

// Request a token from the web, then returns the retrieved token.
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
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	saveToken(tokFile, tok)
	return config.Client(context.Background(), tok)
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

var clients = map[string]*http.Client{}
var configs = map[string]*oauth2.Config{}
var services = map[string]*drive.Service{}

func runHttpServer() {
	http.HandleFunc("/api/createStorage", func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("userId")
		b, err := os.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}
		config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
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
	http.HandleFunc("/api/authorizeStorage", func(w http.ResponseWriter, r *http.Request) {
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
	http.HandleFunc("/api/listFiles", func(w http.ResponseWriter, req *http.Request) {
		userId := req.Header.Get("userId")
		srv := services[userId]
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
	http.HandleFunc("/api/upload", func(w http.ResponseWriter, req *http.Request) {
		userId := req.Header.Get("userId")
		file := req.Header.Get("file")
		srv := services[userId]
		filename := req.Header.Get("fileName")
		fileReader, err := os.Open("/app/input/" + file)
		if err != nil {
			log.Println(err)
		}
		defer fileReader.Close()
		info, err := fileReader.Stat()
		if err != nil {
			log.Println(err)
		}
		size := info.Size()
		res, err := srv.Files.Create(
			&drive.File{
				Name: filename,
			},
		).Media(fileReader, googleapi.ChunkSize(int(size))).Do()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%s\n", res.Id)
		resp, _ := json.Marshal(map[string]any{"success": true, "fileId": res.Id})
		w.Write(resp)
		os.Remove("/app/input/" + file)
	})
	http.HandleFunc("/api/download", func(w http.ResponseWriter, req *http.Request) {
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
		resp, _ := json.Marshal(map[string]any{"success": true, "data": string(data)})
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
	flag.Parse()

	if command == "adminInit" {
		runHttpServer()
	} else if command == "createStorage" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/createStorage", bytes.NewBuffer([]byte("{}")))
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
		req.Header.Set("fileName", file)
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
