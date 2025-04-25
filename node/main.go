package main

import (
	"log"
	_ "net/http/pprof"
	"os"

	cmd "kasper/cmd/babble/commands"
)

import "C"

//export elpisCallback
func elpisCallback(dataRaw *C.char) *C.char {
	return C.CString(cmd.KasperApp.Tools().Elpis().ElpisCallback(C.GoString(dataRaw)))
}

//export wasmCallback
func wasmCallback(dataRaw *C.char) *C.char {
	return C.CString(cmd.KasperApp.Tools().Wasm().WasmCallback(C.GoString(dataRaw)))
}

func main() {

	rootCmd := cmd.RootCmd

	rootCmd.AddCommand(
		cmd.VersionCmd,
		cmd.NewKeygenCmd(),
		cmd.NewRunCmd())

	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
