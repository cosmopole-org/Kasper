package main

import (
	"crypto/x509"
	"encoding/pem"
	kasper "kasper/src/shell"
	plugger_api "kasper/src/shell/api/main"
	"os"
	"strconv"

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
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		panic(err)
	}
	app := kasper.NewApp(ownerId, privateKey)

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
			"baseDbPath": os.Getenv("BASE_DB_PATH"),
			"federationPort": int(federationPort),
			"blockchainPort": int(blockchainPort),
			"pointLogsDb": os.Getenv("POINT_LOGS_DB"),
		},
	)
	
	portStr := os.Getenv("CLIENT_API_PORT")
	port, _ := strconv.ParseInt(portStr, 10, 64)
	plugger_api.PlugAll(app)

	app.Tools().Network().Run(
		map[string]int{
			"tcp": int(port),
		},
	)

	<-exit
}
