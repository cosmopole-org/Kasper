package sdk

import (
	model "applet/src/models"
)

var RunFunc func(model.Send)

func SetRunFunc(cb func(model.Send)) {
	RunFunc = cb
}

//export run
func run(a int64) int64 {

	Console.Log("parsing input...")
	signal := ParseArgs(a)

	RunFunc(signal)

	return 0
}
