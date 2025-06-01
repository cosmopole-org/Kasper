package main

import (
	"os"
	"strconv"
	plugger_api "kasper/src/shell/api/main"
	plugger_machiner "kasper/src/shell/machiner/main"
	"github.com/joho/godotenv"
	kasper "kasper/src/shell"

	_ "net/http/pprof"
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

	app := kasper.NewApp(kasper.Config{
		Id: os.Getenv("ORIGIN"),
	})

	KasperApp = app

	federationPort, err := strconv.ParseInt(os.Getenv("FEDERATION_API_PORT"), 10, 64)
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
			"pointLogsDb": os.Getenv("POINT_LOGS_DB"),
		},
	)
	
	portStr := os.Getenv("CLIENT_API_PORT")
	port, _ := strconv.ParseInt(portStr, 10, 64)
	plugger_api.PlugAll(app)
	plugger_machiner.PlugAll(app)

	app.Tools().Network().Run(
		map[string]int{
			"tcp": int(port),
		},
	)

	<-exit
}
