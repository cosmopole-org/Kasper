package model

type WasmInput struct {
	Data string
}

func (d WasmInput) GetPointId() string {
	return ""
}

func (d WasmInput) Origin() string {
	return ""
}