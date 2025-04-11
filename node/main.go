package main

import (
	"log"
	_ "net/http/pprof"
	"os"

	cmd "kasper/cmd/babble/commands"
	"kasper/src/abstract"
	tb "kasper/src/shell/layer2/model"
)

import "C"

//export elpisCallback
func elpisCallback(dataRaw *C.char) *C.char {
	return C.CString(abstract.UseToolbox[*tb.ToolboxL2](cmd.KasperApp.Get(2).Tools()).Elpis().ElpisCallback(C.GoString(dataRaw)))
}

//export wasmCallback
func wasmCallback(dataRaw *C.char) *C.char {
	return C.CString(abstract.UseToolbox[*tb.ToolboxL2](cmd.KasperApp.Get(2).Tools()).Wasm().WasmCallback(C.GoString(dataRaw)))
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
