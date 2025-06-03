package firectl

import (
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
)

type VM struct {
	Machine  *firecracker.Machine
	Terminal ITerminalManager
	SigCh    chan int
}

type ITerminalManager interface {
	Start()
	Stop()
	GetOutput() string
	SendCommand(string)
	RegisterListener() <-chan string
	RemoveListener(ch <-chan string)
}

type IFirectl interface {
	StopVm(id string)
	GetVm(id string) *VM
	RunVm(id string, terminal chan string)
}
