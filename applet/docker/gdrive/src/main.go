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

func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		getTokenFromWeb(config)
		return nil
	} else {
		return config.Client(context.Background(), tok)
	}
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) {
	config.RedirectURL = "http://localhost:8080"
	config.Scopes = append(config.Scopes, "https://www.googleapis.com/auth/drive.file")
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)
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
		if client != nil {
			clients[userId] = client
			ctx := context.Background()
			srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
			if err != nil {
				log.Fatalf("Unable to retrieve Drive client: %v", err)
			}
			services[userId] = srv
		}
		configs[userId] = config
		res, _ := json.Marshal(map[string]any{"success": true})
		w.Write(res)
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
		if len(r.Files) == 0 {
			fmt.Println("No files found.")
		} else {
			for _, i := range r.Files {
				fmt.Printf("%s (%s)\n", i.Name, i.Id)
			}
		}
		res, _ := json.Marshal(map[string]any{"success": true})
		w.Write(res)
	})
	http.HandleFunc("/api/upload", func(w http.ResponseWriter, req *http.Request) {
		userId := req.Header.Get("userId")
		body := req.Body
		arr := make([]byte, 1024)
		length, err := body.Read(arr)
		if err != nil {
			log.Println(err)
		}
		srv := services[userId]
		filename := req.Header.Get("fileName")
		res, err := srv.Files.Create(
			&drive.File{
				Name: filename,
			},
		).Media(bytes.NewBuffer(arr[:length]), googleapi.ChunkSize(int(length))).Do()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%s\n", res.Id)
		resp, _ := json.Marshal(map[string]any{"success": true})
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
			panic(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
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
		fmt.Println(string(body))
	} else if command == "listFiles" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/listFiles", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("userId", userId)
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
	} else if command == "upload" {
		req, _ := http.NewRequest("POST", "http://localhost:3000/api/upload", bytes.NewBuffer([]byte("hello keyhan !")))
		req.Header.Set("userId", userId)
		req.Header.Set("fileName", "test.txt")
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
	}
}
