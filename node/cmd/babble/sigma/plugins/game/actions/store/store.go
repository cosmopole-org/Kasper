package actions_player

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kasper/cmd/babble/sigma/abstract"
	model "kasper/cmd/babble/sigma/api/model"
	"kasper/cmd/babble/sigma/layer1/adapters"
	states "kasper/cmd/babble/sigma/layer1/module/state"
	game_inputs_store "kasper/cmd/babble/sigma/plugins/game/inputs/store"
	game_model "kasper/cmd/babble/sigma/plugins/game/model"
	"kasper/cmd/babble/sigma/utils/crypto"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&game_model.Payment{})
}

// Buy /store/buy check [ true false false ] access [ true false false false POST ]
func (a *Actions) Buy(s abstract.IState, input game_inputs_store.BuyInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()
	
	user := model.User{Id: state.Info().UserId()}
	trx.Db().First(&user)
	trx.ClearError()

	market := input.Market
	if market == "" {
		market = "bazar"
	}

	if market == "bazar" {
		url := "https://pardakht.cafebazaar.ir/devapi/v2/auth/token/"
		method := "POST"
		payload := strings.NewReader(`{
    	"grant_type": "refresh_token",
    	"client_id": "FpCfIaOSxJBC8E0EKAt2EISGdozVTZEB8BUYNRov",
    	"client_secret": "2L5ieVOf3sVdwT4WvWWGIrrlKcofzQ7s0eKyuguFlPpm8BR9p6lfGMaajYjj",
    	"refresh_token": "wMAzM6mhZr81KjWXrMm1UShLdu9F12"
	}`)
		client := &http.Client{}
		req, err := http.NewRequest(method, url, payload)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		req.Header.Add("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		output := map[string]any{}
		err2 := json.Unmarshal(body, &output)
		if err2 != nil {
			fmt.Println(err)
			return nil, err
		}
		atRaw, ok := output["access_token"]
		if !ok {
			err := errors.New("access token not returned")
			fmt.Println(err)
			return nil, err
		}
		accessToken := atRaw.(string)

		url2 := "https://pardakht.cafebazaar.ir/devapi/v2/api/validate/com.midopia.hokmraan/inapp/" + input.Product + "/purchases/" + input.PurchaseToken + "/?access_token=" + accessToken
		method2 := "GET"
		req2, err := http.NewRequest(method2, url2, nil)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		res2, err := client.Do(req2)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		defer res2.Body.Close()
		body2, err := ioutil.ReadAll(res2.Body)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		output2 := map[string]any{}
		err3 := json.Unmarshal(body2, &output2)
		if err3 != nil {
			fmt.Println(err3)
			return nil, err3
		}
		_, ok2 := output2["purchaseTime"]
		if !ok2 {
			err := errors.New("purchase not found")
			fmt.Println(err)
			return nil, err
		}

		url3 := "https://pardakht.cafebazaar.ir/devapi/v2/api/consume/com.midopia.hokmraan/purchases/?access_token=" + accessToken
		method3 := "POST"
		payload3 := strings.NewReader(`{
    		"token": "` + input.PurchaseToken + `"
		}`)
		req3, err := http.NewRequest(method3, url3, payload3)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		req3.Header.Add("Content-Type", "application/json")
		res3, err := client.Do(req3)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		defer res3.Body.Close()
		if res3.StatusCode != http.StatusOK {
			errC := errors.New("consuming bazar error")
			fmt.Println(errC)
			return nil, errC
		}

		body3, err := ioutil.ReadAll(res3.Body)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		log.Println(string(body3))
	} else if market == "myket" {
		url := "https://developer.myket.ir/api/partners/applications/com.midopia.hokmraan/purchases/products/" + input.Product + "/verify"
		method := "POST"
		payload := strings.NewReader(`{
    		"tokenId": "` + input.PurchaseToken + `"
		}`)
		client := &http.Client{}
		req, err := http.NewRequest(method, url, payload)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-Access-Token", "9132a90e-20cf-4500-91b7-26f31a0417be")
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		output := map[string]any{}
		err2 := json.Unmarshal(body, &output)
		if err2 != nil {
			fmt.Println(err)
			return nil, err
		}
		o, oko := output["purchaseState"]
		if !oko || (o.(float64) != 0) {
			err := errors.New("unsucessful myket purchase")
			fmt.Println(err)
			return nil, err
		}

		url2 := "https://developer.myket.ir/api/partners/applications/com.midopia.hokmraan/purchases/products/" + input.Product + "/tokens/" + input.PurchaseToken + "/consume"
		method2 := "PUT"
		req2, err := http.NewRequest(method2, url2, nil)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		req2.Header.Add("Content-Type", "application/json")
		req2.Header.Add("X-Access-Token", "9132a90e-20cf-4500-91b7-26f31a0417be")
		res2, err := client.Do(req2)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		defer res2.Body.Close()
		body2, err := ioutil.ReadAll(res2.Body)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		output2 := map[string]any{}
		err3 := json.Unmarshal(body2, &output2)
		if err3 != nil {
			fmt.Println(err3)
			return nil, err3
		}
		o2, oko2 := output2["code"]
		if !oko2 || (o2.(float64) != 200) {
			errC := errors.New("consuming myket error")
			fmt.Println(errC)
			return nil, errC
		}
	}

	effects := map[string]float64{}
	userData := map[string]interface{}{}

	meta := game_model.Meta{Id: input.GameKey + "::buy"}
	err4 := trx.Db().First(&meta).Error
	if err4 != nil {
		return nil, err4
	}
	val := meta.Data[input.Product].(string)
	data := strings.Split(val, ".")
	for i := range data {
		if i%2 == 0 {
			number, err5 := strconv.ParseFloat(data[i+1], 64)
			if err5 != nil {
				fmt.Println(err5)
				continue
			}
			effects[data[i]] = number
		}
	}
	trx.ClearError()
	gameDataStr := ""
	trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", state.Info().UserId()).First(&gameDataStr)
	trx.ClearError()
	err6 := json.Unmarshal([]byte(gameDataStr), &userData)
	if err6 != nil {
		log.Println(err6)
		return nil, err6
	}

	for k, v := range effects {
		if v == 0 {
			continue
		}
		timeKey := "last" + (strings.ToUpper(string(k[0])) + k[1:]) + "Buy"
		now := int64(time.Now().UnixMilli())
		oldValRaw, ok := userData[k]
		if !ok {
			continue
		}
		oldVal := oldValRaw.(float64)
		newVal := v + oldVal
		lastBuyTimeRaw, ok2 := userData[timeKey]
		if k == "chat" && ok2 {
			lastBuyTime := lastBuyTimeRaw.(float64)
			if float64(now) < (lastBuyTime + (24 * 60 * 60 * 1000 * oldVal)) {
				newVal = math.Ceil(v + oldVal - ((float64(now) - lastBuyTime) / (24 * 60 * 60 * 1000)))
			} else {
				newVal = v
			}
		}
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &user, "metadata", input.GameKey+"."+k, newVal)
		if err != nil {
			log.Println(err)
			return map[string]any{}, err
		}
		trx.ClearError()
		err2 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", state.Info().UserId()) }, &user, "metadata", input.GameKey+"."+timeKey, now)
		if err2 != nil {
			log.Println(err2)
			return map[string]any{}, err2
		}
		trx.ClearError()
	}
	p := game_model.Payment{Market: market, Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: state.Info().UserId(), Product: input.Product, GameKey: input.GameKey, Time: time.Now().UnixMilli()}
	trx.Db().Create(&p)
	return map[string]any{}, nil
}
