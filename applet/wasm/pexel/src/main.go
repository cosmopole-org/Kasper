package main

import (
	model "applet/src/models"
	api "applet/src/sdk"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
)

const API_KEY = "nuePWQkpdcb94U2CR81DUb9w7wsvuD5IprRyx0yaUYiSJQHPJoWcuhIj"

func Run(signal model.Send) {

	api.Init()

	trx := &api.Trx[api.MyDb]{
		Db:       api.NewMyDb(),
		Chain:    &api.Chain{},
		Offchain: &api.OffChain{},
		Signaler: &api.Signaler{},
		Network:  &api.NetHttp{},
	}

	input := map[string]any{}
	err := json.Unmarshal([]byte(signal.Data), &input)
	if err != nil {
		api.Console.Log(err.Error())
	}
	actRaw, ok := input["type"]
	if !ok {
		trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 1}, true)
	}
	act, ok := actRaw.(string)
	if !ok {
		trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"success": false, "errCode": 2}, true)
	}

	switch act {
	case "adminStart":
		{
			trx.Db.BaseDB.Put("shouldRun", []byte("true"))
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"message": "pexel app started."}, true)
			points := trx.Db.Points.Read("all", "", "")
			for _, point := range points {
				res := trx.Network.Request("GET", "https://api.pexels.com/v1/curated?per_page=1", map[string]string{"Authorization": API_KEY}, map[string]any{})
				m := map[string]any{}
				j, _ := base64.StdEncoding.DecodeString(res)
				json.Unmarshal([]byte(j), &m)
				url := m["photos"].([]any)[0].(map[string]any)["src"].(map[string]any)["large"].(string)
				data := trx.Network.Request("GET", url, map[string]string{}, map[string]any{})
				inp := model.UploadPointEntityInput{
					EntityId: "background",
					PointId:  point.Id,
					Data:     data,
				}
				trx.Offchain.SubmitBaseRequest(signal.Point.Id, "/storage/uploadPointEntity", "", "", "", inp)
			}
			trx.Offchain.PlantRewoke(120, signal.Point.Id, map[string]any{
				"type": "updateWallpapers",
			})
			break
		}
	case "adminStop":
		{
			trx.Db.BaseDB.Put("shouldRun", []byte("false"))
			trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"message": "pexel app stopped."}, true)
			break
		}
	case "updateWallpapers":
		{
			if string(trx.Db.BaseDB.Get("shouldRun")) == "true" {
				points := trx.Db.Points.Read("all", "", "")
				for _, point := range points {
					res := trx.Network.Request("GET", "https://api.pexels.com/v1/curated?per_page=50", map[string]string{"Authorization": API_KEY}, map[string]any{})
					m := map[string]any{}
					j, _ := base64.StdEncoding.DecodeString(res)
					json.Unmarshal([]byte(j), &m)
					url := m["photos"].([]any)[rand.Intn(len(m["photos"].([]any)))].(map[string]any)["src"].(map[string]any)["large"].(string)
					data := trx.Network.Request("GET", url, map[string]string{}, map[string]any{})
					inp := model.UploadPointEntityInput{
						EntityId: "background",
						PointId:  point.Id,
						Data:     data,
					}
					trx.Offchain.SubmitBaseRequest(signal.Point.Id, "/storage/uploadPointEntity", "", "", "", inp)
				}
				trx.Offchain.PlantRewoke(120, signal.Point.Id, map[string]any{
					"type": "updateWallpapers",
				})
			}
		}
	case "textMessage":
		{
			text := input["text"].(string)
			if strings.Contains(text, "@pexel") {
				if strings.Trim(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, "@pexel", ""), "\n", ""), "\t", ""), " ") == "/activate" {
					trx.Db.Points.CreateAndInsert(&api.Point{
						Id: signal.Point.Id,
					})
					trx.Signaler.Answer(signal.Point.Id, signal.User.Id, map[string]any{"type": "textMessage", "text": "pexel activated"}, false)

					point := trx.Db.Points.FindById(signal.Point.Id)
					res := trx.Network.Request("GET", "https://api.pexels.com/v1/curated?per_page=1", map[string]string{"Authorization": API_KEY}, map[string]any{})
					m := map[string]any{}
					j, _ := base64.StdEncoding.DecodeString(res)
					json.Unmarshal([]byte(j), &m)
					url := m["photos"].([]any)[0].(map[string]any)["src"].(map[string]any)["large"].(string)
					data := trx.Network.Request("GET", url, map[string]string{}, map[string]any{})
					inp := model.UploadPointEntityInput{
						EntityId: "background",
						PointId:  point.Id,
						Data:     data,
					}
					trx.Offchain.SubmitBaseRequest(signal.Point.Id, "/storage/uploadPointEntity", "", "", "", inp)
				}
			}
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
