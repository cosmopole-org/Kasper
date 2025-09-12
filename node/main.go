package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"kasper/src/abstract/models/action"
	kasper "kasper/src/shell"
	plugger_api "kasper/src/shell/api/main"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
)

import "C"

var KasperApp kasper.Kasper

//export elpisCallback
func elpisCallback(dataRaw *C.char) *C.char {
	return C.CString(KasperApp.Tools().Elpis().ElpisCallback(C.GoString(dataRaw)))
}

//export wasmCallback
func wasmCallback(dataRaw *C.char) *C.char {
	return C.CString(KasperApp.Tools().Wasm().WasmCallback(C.GoString(dataRaw)))
}

var exit = make(chan int, 1)

func main() {

	err2 := godotenv.Load()
	if err2 != nil {
		panic(err2)
	}

	ownerId := os.Getenv("OWNER_ID")
	privateKeyBlock, _ := pem.Decode([]byte(os.Getenv("OWNER_PRIVATE_KEY")))
	privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		panic(err)
	}
	app := kasper.NewApp(os.Getenv("ORIGIN"), ownerId, privateKey.(*rsa.PrivateKey))

	KasperApp = app

	federationPort, err := strconv.ParseInt(os.Getenv("FEDERATION_API_PORT"), 10, 64)
	if err != nil {
		panic(err)
	}

	blockchainPort, err := strconv.ParseInt(os.Getenv("BLOCKCHAIN_API_PORT"), 10, 64)
	if err != nil {
		panic(err)
	}

	app.Load(
		[]string{
			"keyhan",
		},
		map[string]interface{}{
			"storageRoot":  os.Getenv("STORAGE_ROOT_PATH"),
			"appletDbPath": os.Getenv("APPLET_DB_PATH"),
			"baseDbPath":   os.Getenv("BASE_DB_PATH"),
			"pointLogsDb":  os.Getenv("POINT_LOGS_DB"),
		},
	)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-c
		app.Close()
		os.Exit(0)
	}()

	plugger_api.PlugAll(app, map[string]map[string]action.ExtendedField{
		"user": {
			"name": {
				Name:         "name",
				Path:         "metadata.public.profile",
				Type:         "string",
				Default:      "Anonymous User",
				Required:     true,
				Searchable:   true,
				PrimaryProp: true,
			},
			"avatar": {
				Name:         "avatar",
				Path:         "metadata.public.profile",
				Type:         "string",
				Default:      "avatar",
				Required:     true,
				Searchable:   false,
				PrimaryProp: true,
			},
			"bio": {
				Name:         "bio",
				Path:         "metadata.public.profile",
				Type:         "string",
				Default:      "I'm a DecillionAI User",
				Required:     true,
				Searchable:   false,
				PrimaryProp: false,
			},
			"location": {
				Name:         "location",
				Path:         "metadata.public.profile",
				Type:         "string",
				Default:      "DecillionAI Land",
				Required:     true,
				Searchable:   false,
				PrimaryProp: false,
			},
		},
		"point": {
			"title": {
				Name:        "title",
				Path:        "metadata.public.profile",
				Type:        "string",
				Default:     "Untitled Point",
				Required:    true,
				Searchable:  true,
				PrimaryProp: true,
			},
			"avatar": {
				Name:         "avatar",
				Path:         "metadata.public.profile",
				Type:         "string",
				Default:      "avatar",
				Required:     true,
				Searchable:   false,
				PrimaryProp: true,
			},
		},
	})

	portStr := os.Getenv("CLIENT_TCP_API_PORT")
	port, _ := strconv.ParseInt(portStr, 10, 64)
	portStr2 := os.Getenv("CLIENT_WS_API_PORT")
	port2, _ := strconv.ParseInt(portStr2, 10, 64)

	app.Tools().Network().Run(
		map[string]int{
			"tcp":   int(port),
			"ws":    int(port2),
			"fed":   int(federationPort),
			"chain": int(blockchainPort),
		},
	)

	<-exit
}
