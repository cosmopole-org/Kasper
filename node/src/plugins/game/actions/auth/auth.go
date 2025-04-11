package actions_player

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kasper/src/abstract"
	moduleactormodel "kasper/src/core/module/actor/model"
	game_inputs_auth "kasper/src/plugins/game/inputs/auth"
	game_model "kasper/src/plugins/game/model"
	game_outputs_auth "kasper/src/plugins/game/outputs/auth"
	inputs_users "kasper/src/shell/api/inputs/users"
	"kasper/src/shell/api/model"
	outputs_users "kasper/src/shell/api/outputs/users"
	"kasper/src/shell/layer1/adapters"
	states "kasper/src/shell/layer1/module/state"
	"kasper/src/shell/layer1/module/toolbox"
	"kasper/src/shell/utils/crypto"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return s.AutoMigrate(&game_model.Meta{})
}

type IpTable struct {
	table   map[string]int32
	blocked map[string]int64
	mux     sync.Mutex
}

var ipStore = &IpTable{table: map[string]int32{}, blocked: map[string]int64{}}

var (
	zeroDialer net.Dialer
)

var netClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return zeroDialer.DialContext(ctx, "tcp4", addr)
		},
	},
	Timeout: time.Duration(1000) * time.Second,
}

var gamesMetadataTemplate = map[string]map[string]any{
	"hokm": map[string]any{
		"profile": map[string]any{
			"name":   "",
			"avatar": -1,
		},
		"chat":         0,
		"lastChatBuy":  0,
		"banned":       false,
		"registerDate": 0,
		"gem":          "var",
		"games":        0,
		"wins":         0,
		"level":        `func(js(level))`,
		"xp":           0,
		"maxXp":        `func(js(maxXp))`,
		"league":       `func(js(league))`,
		"chatCharge":   "func(js(chatCharge))",
		"rank":         []string{},
		"purchase":     []string{},
		"win1":         int64(0),
		"win2":         int64(0),
		"win3":         int64(0),
		"friends":      "func(count(friends))",
		"friendReqs":   "func(count(friendReqs))",
		"unreadChats":  "func(count(unreadChats))",
		"score":        0,
		"newlyCreated": true,
	},
}

func prepareGameData(trx adapters.ITrx, user *model.User, name string) error {
	for key, game := range gamesMetadataTemplate {
		gameDataStr := ""
		err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", key)).Where("id = ?", user.Id).First(&gameDataStr).Error
		if err != nil || gameDataStr == "" {
			trx.ClearError()
			err2 := adapters.UpdateJson(func() *gorm.DB {
				log.Println("user id : [", user.Id, "]")
				return trx.Db().Model(&model.User{}).Where("id = ?", user.Id)
			}, user, "metadata", key, game)
			if err2 != nil {
				log.Println(err2)
				return err2
			}
			title := name
			if title == "" {
				rawNum := rand.Intn(9999)
				num := fmt.Sprintf("%d", rawNum)
				for i := 0; i < (4 - len(num)); i++ {
					num = "0" + num
				}
				title = "user" + num
			}
			err3 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", user.Id) }, user, "metadata", key+".profile.name", title)
			if err3 != nil {
				log.Println(err3)
				return err3
			}
			meta := game_model.Meta{Id: key}
			trx.Db().First(&meta)
			for k, v := range game {
				val, ok := v.(string)
				if ok {
					if val == "var" {
						err4 := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", user.Id) }, user, "metadata", key+"."+k, meta.Data["start"+k])
						if err4 != nil {
							log.Println(err4)
							return err4
						}
					}
				}
			}
		}
	}
	return nil
}

// Login /auth/login check [ false false false ] access [ true false false false POST ]
func (a *Actions) Login(s abstract.IState, input game_inputs_auth.LoginInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	var ip = state.Dummy()
	if ip != "" {
		ipStore.mux.Lock()
		if ipStore.table[ip] > 5 {
			if (time.Now().UnixMilli() - ipStore.blocked[ip]) > (1 * 60 * 1000) {
				ipStore.table[ip] = 0
			} else {
				ipStore.mux.Unlock()
				return nil, errors.New("you are temporary blocked for reaching register try count for ip:" + ip)
			}
		} else {
			ipStore.table[ip] = ipStore.table[ip] + 1
		}
		ipStore.blocked[ip] = time.Now().UnixMilli()
		ipStore.mux.Unlock()
	}
	var email = ""
	var name = ""
	if input.EmailToken == nil || (len(*input.EmailToken) == 0) {
		email = crypto.SecureUniqueString()
	} else {
		var resp, err = netClient.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + (*input.EmailToken))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return nil, err2
		}
		result := make(map[string]interface{})
		err3 := json.Unmarshal(body, &result)
		if err3 != nil {
			return nil, err3
		}
		if result["email"] != nil {
			email = result["email"].(string)
		} else {
			return nil, errors.New("access denied")
		}
		if result["given_name"] != nil {
			name = result["given_name"].(string)
		} else {
			return nil, errors.New("access denied")
		}

	}
	trx := state.Trx()

	username := email + "@" + a.Layer.Core().Id()
	if (input.SessionToken != nil) && (len(*input.SessionToken) > 0) {
		var session = model.Session{}
		err := trx.Db().Where("token = ?", *input.SessionToken).First(&session).Error
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if (input.EmailToken != nil) && (len(*input.EmailToken) > 0) {
			trx.ClearError()
			var user = model.User{}
			err2 := trx.Db().Where("username = ?", username).First(&user).Error
			if err2 != nil {
				log.Println(err2)
				trx.ClearError()
				user = model.User{Id: session.UserId}
				err3 := trx.Db().First(&user).Error
				if err3 != nil {
					log.Println(err3)
					trx.ClearError()
				}
				user.Username = username
				err4 := trx.Db().Save(&user).Error
				if err4 != nil {
					log.Println(err4)
					return nil, err4
				}
			} else {
				trx.ClearError()
				if user.Id != session.UserId {
					session = model.Session{}
					trx.Db().Where("user_id = ?", user.Id).First(&session)
					prepareGameData(trx, &user, name)
					return game_outputs_auth.LoginOutput{Username: user.Username, Token: session.Token}, nil
				}
			}
			prepareGameData(trx, &user, name)
			return game_outputs_auth.LoginOutput{Username: user.Username, Token: session.Token}, nil
		} else {
			trx.ClearError()
			user := model.User{Id: session.UserId}
			err2 := trx.Db().First(&user).Error
			log.Println(session)
			log.Println(user)
			if err2 != nil {
				log.Println(err2)
				return nil, err2
			}
			prepareGameData(trx, &user, name)
			return game_outputs_auth.LoginOutput{Username: user.Username, Token: session.Token}, nil
		}
	} else {
		if (input.EmailToken != nil) && (len(*input.EmailToken) > 0) {
			var user = model.User{}
			err2 := trx.Db().Where("username = ?", username).First(&user).Error
			if err2 != nil {
				log.Println(err2)
				trx.ClearError()

				_, res, err := a.Layer.Core().Get(1).Actor().FetchAction("/users/login").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo("", "", "", ""), state.Trx(), state.Dummy()),
					inputs_users.LoginInput{
						Username: email,
					})
				if err != nil {
					return nil, err
				}
				u := res.(outputs_users.LoginOutput).User
				prepareGameData(trx, &u, name)
				return game_outputs_auth.LoginOutput{Username: u.Username, Token: res.(outputs_users.LoginOutput).Session.Token}, nil
			} else {
				trx.ClearError()
				session := model.Session{}
				err3 := trx.Db().Where("user_id = ?", user.Id).First(&session).Error
				if err3 != nil {
					log.Println(err3)
					return game_outputs_auth.LoginOutput{}, err3
				}
				e := prepareGameData(trx, &user, name)
				if e != nil {
					return nil, e
				}
				return game_outputs_auth.LoginOutput{Username: user.Username, Token: session.Token}, nil
			}
		} else {
			_, res, err := a.Layer.Core().Get(1).Actor().FetchAction("/users/login").Act(a.Layer.Sb().NewState(moduleactormodel.NewInfo("", "", "", ""), state.Trx(), state.Dummy()),
				inputs_users.LoginInput{
					Username: email,
				})
			if err != nil {
				return nil, err
			}
			u := res.(outputs_users.LoginOutput).User
			log.Println(u)
			return func() (any, error) {
				e := abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools()).Storage().DoTrx(func(trx adapters.ITrx) error {
					return prepareGameData(trx, &u, name)
				})
				if e != nil {
					return nil, e
				}
				return game_outputs_auth.LoginOutput{Username: u.Username, Token: res.(outputs_users.LoginOutput).Session.Token}, nil
			}, nil
		}
	}
}
