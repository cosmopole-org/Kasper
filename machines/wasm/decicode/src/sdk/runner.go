package sdk

import (
	model "applet/src/models"
	"encoding/json"
	"maps"
)

var RunFunc func(model.Send, string, map[string]any)

func SetRunFunc(cb func(model.Send, string, map[string]any)) {
	RunFunc = cb
}

//export run
func run(a int64) int64 {

	Console.Log("parsing input...")
	signal := ParseArgs(a)

	input := map[string]any{}
	e := json.Unmarshal([]byte(signal.Data), &input)
	if e != nil {
		Console.Log(e.Error())
		return 0
	}
	typ := input["type"].(string)
	args := input
	delete(args, "type")

	if typ == "execute" {
		typ = input["name"].(string)
		args = input["args"].(map[string]any)
		p := map[string]any{
			"type": typ,
		}
		maps.Copy(p, args)
		b, _ := json.Marshal(p)
		signal.Data = string(b)
	}

	RunFunc(signal, typ, args)

	return 0
}
