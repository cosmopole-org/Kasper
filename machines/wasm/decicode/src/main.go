package main

import (
	model "applet/src/models"
	input_model_points "applet/src/models/inputs/points"
	output_model_points "applet/src/models/outputs/points"
	api "applet/src/sdk"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func Run(signal model.Send) {

	api.Init()

	isOnChain := signal.Point.Id == ""

	trx := &api.Trx[api.MyDb]{
		Db:       api.NewMyDb(),
		Chain:    &api.Chain{},
		Offchain: &api.OffChain{},
		Signaler: &api.Signaler{},
		Network:  &api.NetHttp{},
	}

	input := map[string]any{}
	err := json.Unmarshal(bytes.Trim([]byte(signal.Data), "\x00"), &input)
	if err != nil {
		api.Console.Log(err.Error())
	}
	actRaw, ok := input["type"]
	if !ok && !isOnChain {
		trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 1}, true)
	}
	act, ok := actRaw.(string)
	if !ok && !isOnChain {
		trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 2}, true)
	}

	switch act {
	case "initWorkspace":
		{
			if wId := trx.Db.BaseDB.Get(signal.Point.Id); len(wId) == 0 {
				inp := input_model_points.CreateInput{
					Tag:      "workspace",
					IsPublic: false,
					PersHist: true,
					ParentId: signal.Point.Id,
					Orig:     "",
					Members: map[string]bool{
						signal.User.Id: true,
						"41@global":    true,
					},
					Metadata: map[string]any{
						"title":  "decicode-" + signal.User.Id,
						"avatar": "avatar",
					},
				}
				res := trx.Offchain.SubmitBaseRequest(signal.Point.Id, "/points/create", "", "", "", inp)
				output := output_model_points.CreateOutput{}
				json.Unmarshal(res, &output)
				point := output.Point
				p := &api.Point{Id: point.Id, CreatorId: signal.User.Id}
				trx.Db.Points.CreateAndInsert(p)
				trx.Db.BaseDB.Put(signal.Point.Id, []byte(point.Id))
				trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": p}, true)
			} else {
				point := trx.Db.Points.FindById(string(wId))
				trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": point}, true)
			}
			break
		}
	case "files.purge":
		{
			point := trx.Db.Points.FindById(signal.Point.Id)
			point.Docs.DeleteAll()
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": []api.Doc{}}, true)
			break
		}
	case "files.create":
		{
			point := trx.Db.Points.FindById(signal.Point.Id)
			point.Docs.CreateAndInsert(&api.Doc{Id: uuid.NewString(), PointId: point.Id, IsDir: input["isDir"].(bool), CreatorId: signal.User.Id, Title: input["docTitle"].(string), Path: input["docPath"].(string)})
			docs := point.Docs.Read("all", "", "")
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": docs}, true)
			break
		}
	case "files.read":
		{
			point := trx.Db.Points.FindById(signal.Point.Id)
			docs := point.Docs.Read("all", "", "")
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": docs}, true)
			break
		}
	case "files.update":
		{
			point := trx.Db.Points.FindById(signal.Point.Id)
			point.Docs.CreateAndInsert(&api.Doc{Id: input["docId"].(string), PointId: point.Id, CreatorId: signal.User.Id, Title: input["docTitle"].(string), Path: input["docPath"].(string)})
			docs := point.Docs.Read("all", "", "")
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": docs}, true)
			break
		}
	case "files.delete":
		{
			point := trx.Db.Points.FindById(signal.Point.Id)
			point.Docs.DeleteById(input["docId"].(string))
			docs := point.Docs.Read("all", "", "")
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "response", "requestId": input["requestId"], "data": docs}, true)
			break
		}
	}
}

func main() {
	fmt.Println()
	fmt.Println("module starting...")
	fmt.Println()
	api.SetRunFunc(Run)
}
