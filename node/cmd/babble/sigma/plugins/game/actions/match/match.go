package actions_board

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/cmd/babble/sigma/abstract"
	inputs_topics "kasper/cmd/babble/sigma/api/inputs/topics"
	"kasper/cmd/babble/sigma/api/model"
	module_actor_model "kasper/cmd/babble/sigma/core/module/actor/model"
	"kasper/cmd/babble/sigma/layer1/adapters"
	states "kasper/cmd/babble/sigma/layer1/module/state"
	toolbox "kasper/cmd/babble/sigma/layer1/module/toolbox"
	"kasper/cmd/babble/sigma/layer1/tools/signaler"
	statesl3 "kasper/cmd/babble/sigma/layer3/model"
	admin_inputs_board "kasper/cmd/babble/sigma/plugins/admin/inputs/board"
	game_inputs_board "kasper/cmd/babble/sigma/plugins/game/inputs/board"
	game_inputs_match "kasper/cmd/babble/sigma/plugins/game/inputs/match"
	game_model "kasper/cmd/babble/sigma/plugins/game/model"
	game_outputs_match "kasper/cmd/babble/sigma/plugins/game/outputs/match"
	"kasper/cmd/babble/sigma/utils/crypto"
	"kasper/cmd/babble/sigma/utils/future"
	"log"
	"math/rand"
	"strings"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BotProfile struct {
	Name   string
	Avatar string
}

var botUsers = []BotProfile{
	{
		Name:   "ارس",
		Avatar: "6",
	},
	{
		Name:   "مانی",
		Avatar: "4",
	},
	{
		Name:   "شاهین",
		Avatar: "5",
	},
	{
		Name:   "امیر",
		Avatar: "7",
	},
	{
		Name:   "آرش",
		Avatar: "5",
	},
	{
		Name:   "کیهان",
		Avatar: "6",
	},
	{
		Name:   "نسترن",
		Avatar: "0",
	},
	{
		Name:   "مریم",
		Avatar: "2",
	},
	{
		Name:   "لاله",
		Avatar: "1",
	},
	{
		Name:   "آیلین",
		Avatar: "3",
	},
	{
		Name:   "سیما",
		Avatar: "2",
	},
	{
		Name:   "محمد",
		Avatar: "7",
	},
	{
		Name:   "ماکان",
		Avatar: "4",
	},
	{
		Name:   "علیرضا",
		Avatar: "6",
	},
	{
		Name:   "معصومه",
		Avatar: "1",
	},
	{
		Name:   "زهرا",
		Avatar: "2",
	},
	{
		Name:   "زینب",
		Avatar: "3",
	},
	{
		Name:   "آرین",
		Avatar: "7",
	},
	{
		Name:   "آتنا",
		Avatar: "1",
	},
	{
		Name:   "مهسا",
		Avatar: "0",
	},
	{
		Name:   "user8021",
		Avatar: "8",
	},
	{
		Name:   "user8224",
		Avatar: "9",
	},
	{
		Name:   "user9156",
		Avatar: "10",
	},
	{
		Name:   "user1254",
		Avatar: "11",
	},
	{
		Name:   "user7894",
		Avatar: "12",
	},
	{
		Name:   "user5673",
		Avatar: "13",
	},
	{
		Name:   "user9034",
		Avatar: "14",
	},
	{
		Name:   "user6712",
		Avatar: "15",
	},
	{
		Name:   "user9834",
		Avatar: "16",
	},
	{
		Name:   "user3278",
		Avatar: "17",
	},
	{
		Name:   "user9876",
		Avatar: "18",
	},
	{
		Name:   "user4321",
		Avatar: "19",
	},
}

type Actions struct {
	Layer abstract.ILayer
}

type GameBot struct {
	UserId   string
	Oponents int
	Friends  int
}

type LBHolder struct {
	Sharded    bool
	Connectors []LBConnector
}

type LBConnector struct {
	LBLevel string
	Key     string
	Start   float64
}

type GameQueue struct {
	GameKey     string
	Level       string
	Timeout     time.Duration
	PlayerCount int
	HumanCount  int
	GodId       string
	Bot         GameBot
	Poses       []string
	Channel     chan string
	Effects     map[string]string
	LeaderBoard map[string]LBHolder
	Turns       string
	Fee         string
}

var queues *cmap.ConcurrentMap[string, *GameQueue]

var hokmGameUser = model.User{}
var hokmAgentUser = model.User{}

func Install(s adapters.IStorage, a *Actions) error {
	var tb = abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools())
	s.Db().Model(&model.User{}).Where("username = ?", "hokmgame@"+a.Layer.Core().Id()).First(&hokmGameUser)
	s.Db().Model(&model.User{}).Where("username = ?", "hokmagent@"+a.Layer.Core().Id()).First(&hokmAgentUser)
	qs := cmap.New[*GameQueue]()
	queues = &qs
	queues.MSet(map[string]*GameQueue{
		"hokm-friendly": {
			GameKey:     "hokm",
			Level:       "friendly",
			Timeout:     0,
			HumanCount:  2,
			PlayerCount: 2,
			GodId:       hokmGameUser.Id,
			Bot: GameBot{
				UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
			},
			Poses:       []string{"human", "bot", "human|bot", "bot"},
			Channel:     make(chan string, 1),
			Turns:       "turns",
			Fee:         "",
			Effects:     map[string]string{},
			LeaderBoard: map[string]LBHolder{},
		},
		"hokm-1": {
			GameKey:     "hokm",
			Level:       "1",
			Timeout:     10 * time.Second,
			HumanCount:  2,
			PlayerCount: 2,
			GodId:       hokmGameUser.Id,
			Bot: GameBot{
				UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
			},
			Poses:   []string{"human", "bot", "human|bot", "bot"},
			Channel: make(chan string, 1),
			Turns:   "turns",
			Fee:     "gem",
			Effects: map[string]string{
				"gem":   "reward",
				"xp":    "point",
				"score": "score",
			},
			LeaderBoard: map[string]LBHolder{
				"point": {
					Sharded: false,
					Connectors: []LBConnector{
						{
							LBLevel: "1",
							Key:     "tempXp",
						},
						{
							LBLevel: "2",
							Key:     "xp",
						},
					},
				},
				"score": {
					Sharded: true,
					Connectors: []LBConnector{
						{
							LBLevel: "3_1",
							Key:     "score",
							Start:   0,
						},
						{
							LBLevel: "3_2",
							Key:     "score",
							Start:   250,
						},
						{
							LBLevel: "3_3",
							Key:     "score",
							Start:   500,
						},
						{
							LBLevel: "3_4",
							Key:     "score",
							Start:   1000,
						},
						{
							LBLevel: "3_5",
							Key:     "score",
							Start:   2000,
						},
					},
				},
			},
		},
		"hokm-2": {
			GameKey:     "hokm",
			Level:       "2",
			Timeout:     10 * time.Second,
			HumanCount:  2,
			PlayerCount: 2,
			GodId:       hokmGameUser.Id,
			Bot: GameBot{
				UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
			},
			Poses:   []string{"human", "bot", "human|bot", "bot"},
			Channel: make(chan string, 1),
			Turns:   "turns",
			Fee:     "gem",
			Effects: map[string]string{
				"gem":   "reward",
				"xp":    "point",
				"score": "score",
			},
			LeaderBoard: map[string]LBHolder{
				"point": {
					Sharded: false,
					Connectors: []LBConnector{
						{
							LBLevel: "1",
							Key:     "tempXp",
						},
						{
							LBLevel: "2",
							Key:     "xp",
						},
					},
				},
				"score": {
					Sharded: true,
					Connectors: []LBConnector{
						{
							LBLevel: "3_1",
							Key:     "score",
							Start:   0,
						},
						{
							LBLevel: "3_2",
							Key:     "score",
							Start:   250,
						},
						{
							LBLevel: "3_3",
							Key:     "score",
							Start:   500,
						},
						{
							LBLevel: "3_4",
							Key:     "score",
							Start:   1000,
						},
						{
							LBLevel: "3_5",
							Key:     "score",
							Start:   2000,
						},
					},
				},
			},
		},
		"hokm-3": {
			GameKey:     "hokm",
			Level:       "3",
			Timeout:     10 * time.Second,
			HumanCount:  2,
			PlayerCount: 2,
			GodId:       hokmGameUser.Id,
			Bot: GameBot{
				UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
			},
			Poses:   []string{"human", "bot", "human|bot", "bot"},
			Channel: make(chan string, 1),
			Turns:   "turns",
			Fee:     "gem",
			Effects: map[string]string{
				"gem":   "reward",
				"xp":    "point",
				"score": "score",
			},
			LeaderBoard: map[string]LBHolder{
				"point": {
					Sharded: false,
					Connectors: []LBConnector{
						{
							LBLevel: "1",
							Key:     "tempXp",
						},
						{
							LBLevel: "2",
							Key:     "xp",
						},
					},
				},
				"score": {
					Sharded: true,
					Connectors: []LBConnector{
						{
							LBLevel: "3_1",
							Key:     "score",
							Start:   0,
						},
						{
							LBLevel: "3_2",
							Key:     "score",
							Start:   250,
						},
						{
							LBLevel: "3_3",
							Key:     "score",
							Start:   500,
						},
						{
							LBLevel: "3_4",
							Key:     "score",
							Start:   1000,
						},
						{
							LBLevel: "3_5",
							Key:     "score",
							Start:   2000,
						},
					},
				},
			},
		},
		"hokm-4": {
			GameKey:     "hokm",
			Level:       "4",
			Timeout:     10 * time.Second,
			HumanCount:  2,
			PlayerCount: 2,
			GodId:       hokmGameUser.Id,
			Bot: GameBot{
				UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
			},
			Poses:   []string{"human", "bot", "human|bot", "bot"},
			Channel: make(chan string, 1),
			Turns:   "turns",
			Fee:     "gem",
			Effects: map[string]string{
				"gem":   "reward",
				"xp":    "point",
				"score": "score",
			},
			LeaderBoard: map[string]LBHolder{
				"point": {
					Sharded: false,
					Connectors: []LBConnector{
						{
							LBLevel: "1",
							Key:     "tempXp",
						},
						{
							LBLevel: "2",
							Key:     "xp",
						},
					},
				},
				"score": {
					Sharded: true,
					Connectors: []LBConnector{
						{
							LBLevel: "3_1",
							Key:     "score",
							Start:   0,
						},
						{
							LBLevel: "3_2",
							Key:     "score",
							Start:   250,
						},
						{
							LBLevel: "3_3",
							Key:     "score",
							Start:   500,
						},
						{
							LBLevel: "3_4",
							Key:     "score",
							Start:   1000,
						},
						{
							LBLevel: "3_5",
							Key:     "score",
							Start:   2000,
						},
					},
				},
			},
		},
		"hokm-5": {
			GameKey:     "hokm",
			Level:       "5",
			Timeout:     10 * time.Second,
			HumanCount:  2,
			PlayerCount: 2,
			GodId:       hokmGameUser.Id,
			Bot: GameBot{
				UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
			},
			Poses:   []string{"human", "bot", "human|bot", "bot"},
			Channel: make(chan string, 1),
			Turns:   "turns",
			Fee:     "gem",
			Effects: map[string]string{
				"gem":   "reward",
				"xp":    "point",
				"score": "score",
			},
			LeaderBoard: map[string]LBHolder{
				"point": {
					Sharded: false,
					Connectors: []LBConnector{
						{
							LBLevel: "1",
							Key:     "tempXp",
						},
						{
							LBLevel: "2",
							Key:     "xp",
						},
					},
				},
				"score": {
					Sharded: true,
					Connectors: []LBConnector{
						{
							LBLevel: "3_1",
							Key:     "score",
							Start:   0,
						},
						{
							LBLevel: "3_2",
							Key:     "score",
							Start:   250,
						},
						{
							LBLevel: "3_3",
							Key:     "score",
							Start:   500,
						},
						{
							LBLevel: "3_4",
							Key:     "score",
							Start:   1000,
						},
						{
							LBLevel: "3_5",
							Key:     "score",
							Start:   2000,
						},
					},
				},
			},
		},
	})
	for _, q := range queues.Items() {
		if q.Timeout == 0 {
			continue
		}
		future.Async(func() { runQueue(a, q, tb.Signaler()) }, true)
	}
	return nil
}

const memberTemplate = "member::%s::%s::%s"

func startGame(a *Actions, isFriendly bool, sig *signaler.Signaler, trx adapters.ITrx, q *GameQueue, tb *statesl3.ToolboxL3, adminMemberId string, space model.Space, topic model.Topic, gameKey string, players []string, members map[string]model.Member) func() (any, error) {
	var godMember model.Member
	errGod := trx.Db().First(&model.User{Id: q.GodId}).Error
	if errGod != nil {
		log.Println(errGod)
	}
	ti := topic.Id
	if ti == "" {
		ti = "*"
	}
	godMember = model.Member{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: q.GodId, SpaceId: space.Id, TopicId: ti, Metadata: ""}
	errGod2 := trx.Db().Create(&godMember).Error
	if errGod2 != nil {
		log.Println(errGod2)
	}
	trx.Mem().Put(fmt.Sprintf(memberTemplate, godMember.SpaceId, godMember.UserId, godMember.Id), godMember.TopicId)
	tb.Signaler().JoinGroup(space.Id, q.GodId)
	trx.ClearError()
	type playerdata struct {
		UserId         string
		TeamId         string `gorm:"-"`
		Profile        datatypes.JSON
		IsBot          bool         `gorm:"-"`
		Membership     model.Member `gorm:"-"`
		UnmaskedUserId string       `gorm:"-"`
	}
	type playerdatafinal struct {
		UserId         string
		TeamId         string `gorm:"-"`
		Profile        any
		IsBot          bool         `gorm:"-"`
		Membership     model.Member `gorm:"-"`
		UnmaskedUserId string       `gorm:"-"`
	}
	metas := []playerdata{}
	errPD := trx.Db().Model(&model.User{}).Select(
		"id as user_id, "+adapters.BuildJsonFetcher("metadata", gameKey+".profile")+" as profile").
		Where("id in ?", players).
		Where(adapters.BuildJsonFetcher("metadata", gameKey+".profile") + " IS NOT NULL").
		Find(&metas).
		Error
	if errPD != nil {
		log.Println(errPD)
	}
	trx.ClearError()

	playersDict := map[string]string{}
	playersIdToTeamId := map[string]string{}

	humans := []*playerdatafinal{}
	bots := []*playerdatafinal{}

	profileCache := map[string]bool{}

	allPlayerIds := players[:]

	for _, m := range members {
		allPlayerIds = append(allPlayerIds, m.UserId)
		for _, player := range metas {
			if m.UserId == player.UserId {
				gameDataStr := ""
				err := trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", q.GameKey+".profile")).Where("id = ?", m.UserId).First(&gameDataStr).Error
				trx.ClearError()
				if err != nil {
					log.Println(err)
					break
				}
				result := map[string]interface{}{}
				err2 := json.Unmarshal([]byte(gameDataStr), &result)
				if err2 != nil {
					log.Println(err2)
					break
				}
				profileCache[fmt.Sprintf("%d", int(result["avatar"].(float64)))] = true
				break
			}
		}
	}

	type userdata struct {
		Id   string `gorm:"id"`
		Name string `gorm:"name"`
	}

	for _, m := range members {
		isUser := false
		for _, player := range metas {
			if m.UserId == player.UserId {
				isUser = true
				log.Println("human ", m)
				p := &playerdatafinal{UserId: m.UserId, IsBot: false, Profile: map[string]any{}, Membership: m, UnmaskedUserId: m.UserId}
				jsonBytes, err := json.Marshal(player.Profile)
				if err != nil {
					log.Println(err)
				}
				err2 := json.Unmarshal(jsonBytes, &p.Profile)
				if err2 != nil {
					log.Println(err2)
				}
				playersDict[m.UserId] = m.Id
				humans = append(humans, p)
				break
			}
		}
		if !isUser {
			log.Println("bot ", m)
			buIndex := -1
			for (buIndex < 0) || profileCache[botUsers[buIndex].Avatar] {
				buIndex = rand.Intn(len(botUsers))
			}
			profileCache[botUsers[buIndex].Avatar] = true
			bu := botUsers[buIndex]
			title := bu.Name
			us := userdata{}
			trx.Db().Model(&model.User{}).Select("id as id, "+adapters.BuildJsonFetcher("metadata", gameKey+".profile.name")+" as name").Where("id not in ?", allPlayerIds).Where("metadata -> '"+gameKey+"' -> 'profile' ->> 'avatar' = ?", bu.Avatar).Order("RANDOM()").First(&us)
			trx.ClearError()
			if us.Id != "" {
				title = us.Name
				allPlayerIds = append(allPlayerIds, us.Id)
			}
			p := &playerdatafinal{UserId: "b_" + m.Id, IsBot: true, Profile: map[string]any{"name": title, "avatar": bu.Avatar}, Membership: m, UnmaskedUserId: m.UserId}
			playersDict["b_"+m.Id] = m.Id
			bots = append(bots, p)
		}
	}

	finalPlayersList := []*playerdatafinal{}

	for counter, pos := range q.Poses {
		teamId := fmt.Sprintf("%d", (counter%2)+1)
		if pos == "human" && len(humans) > 0 {
			humans[0].TeamId = teamId
			playersIdToTeamId[humans[0].Membership.Id] = teamId
			finalPlayersList = append(finalPlayersList, humans[0])
			if len(humans) > 1 {
				humans = humans[1:]
			} else {
				humans = []*playerdatafinal{}
			}
		} else if pos == "bot" && len(bots) > 0 {
			bots[0].TeamId = teamId
			playersIdToTeamId[bots[0].Membership.Id] = teamId
			finalPlayersList = append(finalPlayersList, bots[0])
			if len(bots) > 1 {
				bots = bots[1:]
			} else {
				bots = []*playerdatafinal{}
			}
		} else if pos == "human|bot" {
			if len(humans) > 0 {
				humans[0].TeamId = teamId
				playersIdToTeamId[humans[0].Membership.Id] = teamId
				finalPlayersList = append(finalPlayersList, humans[0])
				if len(humans) > 1 {
					humans = humans[1:]
				} else {
					humans = []*playerdatafinal{}
				}
			} else if len(bots) > 0 {
				bots[0].TeamId = teamId
				playersIdToTeamId[bots[0].Membership.Id] = teamId
				finalPlayersList = append(finalPlayersList, bots[0])
				if len(bots) > 1 {
					bots = bots[1:]
				} else {
					bots = []*playerdatafinal{}
				}
			}
		}
	}

	type GamePlayer struct {
		UserId   string `json:"userId"`
		MemberId string `json:"memberId"`
		TeamId   string `json:"teamId"`
	}
	gamePlayers := []GamePlayer{}
	for _, player := range finalPlayersList {
		gamePlayers = append(gamePlayers, GamePlayer{UserId: player.UserId, MemberId: playersDict[player.UserId], TeamId: player.TeamId})
		data := map[string]any{"space": space, "topic": topic, "players": finalPlayersList, "godMember": godMember, "myMember": player.Membership}
		sig.SignalUser("/match/join", "", player.UnmaskedUserId, data, true)
	}

	trx.ClearError()

	meta := game_model.Meta{Id: gameKey}
	trx.Db().First(&meta)
	trx.ClearError()

	return func() (any, error) {
		e := tb.Storage().DoTrx(func(trx adapters.ITrx) error {
			s := a.Layer.Sb().NewState(module_actor_model.NewInfo(players[0], space.Id, topic.Id, adminMemberId)).(states.IStateL1)
			s.SetTrx(trx)
			actionSend := a.Layer.Core().Get(1).Actor().FetchAction("/topics/send")
			type gameStartPacket struct {
				Type  string         `json:"type"`
				Value map[string]any `json:"value"`
			}
			gamePlayersStr, errJson := json.Marshal(gameStartPacket{Type: "createGame", Value: map[string]any{"isFriendly": isFriendly, "players": gamePlayers, "level": q.Level, "turns": meta.Data["room"+q.Level+"turns"]}})
			if errJson != nil {
				log.Println(errJson)
			}
			var inp = inputs_topics.SendInput{
				Type:     "single",
				Data:     string(gamePlayersStr),
				RecvId:   godMember.Id,
				SpaceId:  space.Id,
				TopicId:  topic.Id,
				MemberId: adminMemberId,
			}
			_, _, sendErr := actionSend.Act(
				s,
				inp,
			)
			if sendErr != nil {
				log.Println(sendErr)
			}
			return nil
		})
		if e != nil {
			return nil, e
		}
		return map[string]any{}, nil
	}
}

func runQueue(a *Actions, q *GameQueue, sig *signaler.Signaler) {
	gameKey := q.GameKey
	bot := q.Bot
	timeout := q.Timeout
	playerCount := q.PlayerCount
	humanCount := q.HumanCount
	queuePlayers := []string{}
	for {
		errored := false
		select {
		case result := <-q.Channel:
			log.Println("adding user to list of people...")
			queuePlayers = append(queuePlayers, result)
		case <-time.After(timeout):
			if len(queuePlayers) > 0 {
				errored = true
			}
		}
		if (len(queuePlayers) == humanCount) || errored {
			players := []string{}
			players = append(players, queuePlayers...)
			queuePlayers = []string{}
			future.Async(func() {
				if playerCount > len(players) {
					c := len(players)
					for i := 0; i < playerCount-c; i++ {
						players = append(players, bot.UserId)
					}
				}
				for i := 0; i < bot.Oponents; i++ {
					players = append(players, bot.UserId)
				}
				tb := abstract.UseToolbox[*statesl3.ToolboxL3](a.Layer.Tools())
				var callback func() (any, error) = nil
				tb.Storage().DoTrx(func(trx adapters.ITrx) error {
					space := model.Space{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), Tag: "match_" + q.Level, Title: "Game space", Avatar: "0", IsPublic: false}
					err := trx.Db().Create(&space).Error
					if err != nil {
						log.Println(err)
						return err
					}
					member := model.Member{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: players[0], SpaceId: space.Id, TopicId: "*", Metadata: ""}
					err2 := trx.Db().Create(&member).Error
					if err2 != nil {
						log.Println(err2)
						return err
					}
					admin := model.Admin{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: players[0], SpaceId: space.Id, Role: "creator"}
					err3 := trx.Db().Create(&admin).Error
					if err3 != nil {
						log.Println(err3)
						return err
					}
					topic := model.Topic{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), Title: "hall", Avatar: "0", SpaceId: space.Id}
					err4 := trx.Db().Create(&topic).Error
					if err4 != nil {
						log.Println(err4)
						return err
					}
					trx.Mem().Put(fmt.Sprintf("city::%s", topic.Id), topic.SpaceId)
					tb.Signaler().JoinGroup(member.SpaceId, member.UserId)
					trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)

					var members = map[string]model.Member{}
					members[member.Id] = member
					for _, playerId := range players[1:] {
						trx.ClearError()
						var member model.Member
						err := trx.Db().First(&model.User{Id: playerId}).Error
						if err != nil {
							log.Println(err)
						}
						ti := topic.Id
						if ti == "" {
							ti = "*"
						}
						member = model.Member{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: playerId, SpaceId: space.Id, TopicId: ti, Metadata: ""}
						err2 := trx.Db().Create(&member).Error
						if err2 != nil {
							log.Println(err2)
						}
						trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
						tb.Signaler().JoinGroup(member.SpaceId, member.UserId)
						members[member.Id] = member
					}
					trx.ClearError()

					callback = startGame(a, false, sig, trx, q, tb, member.Id, space, topic, gameKey, players, members)

					return nil
				})
				if callback != nil {
					callback()
				}
			}, false)
		}
	}
}

func runUserBuiltQueue(a *Actions, q *GameQueue, sig *signaler.Signaler, starter string) {
	gameKey := q.GameKey
	bot := q.Bot
	timeout := q.Timeout
	playerCount := q.PlayerCount
	humanCount := q.HumanCount
	queuePlayers := []string{starter}
	go func() {
		time.Sleep(timeout)
		q.Channel <- "0"
	}()
	for {
		result := <-q.Channel
		if result == "0" {
			break
		}
		log.Println("adding user to list of people in custom queue...")
		queuePlayers = append(queuePlayers, result)
		if len(queuePlayers) == humanCount {
			players := []string{}
			players = append(players, queuePlayers...)
			queuePlayers = []string{}
			future.Async(func() {
				if playerCount > len(players) {
					c := len(players)
					for i := 0; i < playerCount-c; i++ {
						players = append(players, bot.UserId)
					}
				}
				for i := 0; i < bot.Oponents; i++ {
					players = append(players, bot.UserId)
				}
				tb := abstract.UseToolbox[*statesl3.ToolboxL3](a.Layer.Tools())
				var callback func() (any, error) = nil
				tb.Storage().DoTrx(func(trx adapters.ITrx) error {
					space := model.Space{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), Tag: "match_" + q.Level, Title: "Game space", Avatar: "0", IsPublic: false}
					err := trx.Db().Create(&space).Error
					if err != nil {
						log.Println(err)
						return err
					}
					member := model.Member{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: players[0], SpaceId: space.Id, TopicId: "*", Metadata: ""}
					err2 := trx.Db().Create(&member).Error
					if err2 != nil {
						log.Println(err2)
						return err
					}
					admin := model.Admin{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: players[0], SpaceId: space.Id, Role: "creator"}
					err3 := trx.Db().Create(&admin).Error
					if err3 != nil {
						log.Println(err3)
						return err
					}
					topic := model.Topic{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), Title: "hall", Avatar: "0", SpaceId: space.Id}
					err4 := trx.Db().Create(&topic).Error
					if err4 != nil {
						log.Println(err4)
						return err
					}
					trx.Mem().Put(fmt.Sprintf("city::%s", topic.Id), topic.SpaceId)
					tb.Signaler().JoinGroup(member.SpaceId, member.UserId)
					trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)

					var members = map[string]model.Member{}
					members[member.Id] = member
					for _, playerId := range players[1:] {
						trx.ClearError()
						var member model.Member
						err := trx.Db().First(&model.User{Id: playerId}).Error
						if err != nil {
							log.Println(err)
						}
						ti := topic.Id
						if ti == "" {
							ti = "*"
						}
						member = model.Member{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: playerId, SpaceId: space.Id, TopicId: ti, Metadata: ""}
						err2 := trx.Db().Create(&member).Error
						if err2 != nil {
							log.Println(err2)
						}
						trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
						tb.Signaler().JoinGroup(member.SpaceId, member.UserId)
						members[member.Id] = member
					}
					trx.ClearError()

					callback = startGame(a, false, sig, trx, q, tb, member.Id, space, topic, gameKey, players, members)

					return nil
				})
				if callback != nil {
					callback()
				}
			}, false)
			break
		}
	}
}

type LbShard struct {
	Level string
	Start float64
}

type ShardGroup struct {
	Param  string
	Shards []LbShard
}

var lbShards = map[string]ShardGroup{
	"hokm_3": {
		Param: "score",
		Shards: []LbShard{
			{
				Level: "3_1",
				Start: 0,
			},
			{
				Level: "3_2",
				Start: 250,
			},
			{
				Level: "3_3",
				Start: 500,
			},
			{
				Level: "3_4",
				Start: 1000,
			},
			{
				Level: "3_5",
				Start: 2000,
			},
		},
	},
}

// MyLbShard /board/myLbShard check [ true false false ] access [ true false false false POST ]
func (a *Actions) MyLbShard(s abstract.IState, input game_inputs_match.JoinInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	game, ok := lbShards[input.GameKey+"_"+input.Level]
	if !ok {
		return nil, errors.New("sharded game or level not found")
	}
	p := float64(0)
	trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey+"."+game.Param)).Where("id = ?", state.Info().UserId()).First(&p)
	trx.ClearError()
	shardLevel := ""
	for i := len(game.Shards) - 1; i >= 0; i-- {
		if game.Shards[i].Start <= p {
			shardLevel = game.Shards[i].Level
			break
		}
	}
	return game_outputs_match.MyLbShardOutput{Shard: shardLevel}, nil
}

// Join /match/join check [ true false false ] access [ true false false false POST ]
func (a *Actions) Join(s abstract.IState, input game_inputs_match.JoinInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()
	queueName := input.GameKey + "-" + input.Level
	q, ok := queues.Get(queueName)
	if ok {
		meta := game_model.Meta{Id: input.GameKey}
		trx.Db().First(&meta)
		trx.ClearError()
		if vRaw, ok := meta.Data["maintenance"]; ok {
			if v, ok2 := vRaw.(string); ok2 && (v == "true") {
				return nil, errors.New("in maintenance")
			}
		}
		fee := meta.Data["room"+q.Level+"fee"].(float64)
		gameDataStr := ""
		trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", state.Info().UserId()).First(&gameDataStr)
		trx.ClearError()
		oldVals := map[string]interface{}{}
		err2 := json.Unmarshal([]byte(gameDataStr), &oldVals)
		if err2 != nil {
			log.Println(err2)
		}
		charge := oldVals[q.Fee].(float64)
		if charge >= fee {
			future.Async(func() {
				q.Channel <- state.Info().UserId()
			}, false)
		}
		return game_outputs_match.JoinOutput{}, nil
	} else {
		return nil, errors.New("queue not found")
	}
}

// CreateInHall /match/createInHall check [ true false false ] access [ true false false false POST ]
func (a *Actions) CreateInHall(s abstract.IState, input game_inputs_match.CreateInHallInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()
	queueName := input.GameKey + "-" + input.Level
	q, ok := queues.Get(queueName)
	if ok {
		meta := game_model.Meta{Id: input.GameKey}
		trx.Db().First(&meta)
		trx.ClearError()
		if vRaw, ok := meta.Data["maintenance"]; ok {
			if v, ok2 := vRaw.(string); ok2 && (v == "true") {
				return nil, errors.New("in maintenance")
			}
		}
		fee := meta.Data["room"+input.Level+"fee"].(float64)
		gameDataStr := ""
		trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", state.Info().UserId()).First(&gameDataStr)
		trx.ClearError()
		oldVals := map[string]interface{}{}
		err2 := json.Unmarshal([]byte(gameDataStr), &oldVals)
		if err2 != nil {
			log.Println(err2)
		}
		charge := oldVals[q.Fee].(float64)
		if charge >= fee {
			gq := &GameQueue{
				GameKey:     "hokm",
				Level:       input.Level,
				Timeout:     30 * time.Second,
				HumanCount:  2,
				PlayerCount: 2,
				GodId:       hokmGameUser.Id,
				Bot: GameBot{
					UserId: hokmAgentUser.Id, Oponents: 2, Friends: 1,
				},
				Poses:   []string{"human", "bot", "human|bot", "bot"},
				Channel: make(chan string, 1),
				Turns:   "turns",
				Fee:     "gem",
				Effects: map[string]string{
					"gem":   "reward",
					"xp":    "point",
					"score": "score",
				},
				LeaderBoard: map[string]LBHolder{
					"point": {
						Sharded: false,
						Connectors: []LBConnector{
							{
								LBLevel: "1",
								Key:     "tempXp",
							},
							{
								LBLevel: "2",
								Key:     "xp",
							},
						},
					},
					"score": {
						Sharded: true,
						Connectors: []LBConnector{
							{
								LBLevel: "3_1",
								Key:     "score",
								Start:   0,
							},
							{
								LBLevel: "3_2",
								Key:     "score",
								Start:   250,
							},
							{
								LBLevel: "3_3",
								Key:     "score",
								Start:   500,
							},
							{
								LBLevel: "3_4",
								Key:     "score",
								Start:   1000,
							},
							{
								LBLevel: "3_5",
								Key:     "score",
								Start:   2000,
							},
						},
					},
				},
			}
			qName := "hokm-custom::" + crypto.SecureUniqueString()
			queues.Set(qName, gq)
			future.Async(func() {
				var tb = abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools())
				runUserBuiltQueue(a, gq, tb.Signaler(), state.Info().UserId())
				queues.Remove(qName)
			}, false)
			rooms := []map[string]any{}
			for k, _ := range queues.Items() {
				if strings.HasPrefix(k, "hokm-custom::") {
					rooms = append(rooms, map[string]any{
						"matchId": strings.Split(k, "::")[1],
					})
				}
			}
			var tb = abstract.UseToolbox[toolbox.IToolboxL1](a.Layer.Tools())
			tb.Signaler().SignalGroup("/match/newmatch", state.Info().SpaceId(), rooms, true, []string{})
			return map[string]any{"matchId": qName[len("hokm-"):]}, nil
		}
	}
	return game_outputs_match.JoinOutput{}, nil
}

// GetOpenMatches /match/getOpenMatches check [ true false false ] access [ true false false false POST ]
func (a *Actions) GetOpenMatches(s abstract.IState, input game_inputs_match.GetOpenMatchesInput) (any, error) {
	rooms := []map[string]any{}
	for k, _ := range queues.Items() {
		if strings.HasPrefix(k, input.GameKey + "-" + "custom::") {
			rooms = append(rooms, map[string]any{
				"matchId": k[len(input.GameKey + "-"):],
			})
		}
	}
	return map[string]any{"matches": rooms}, nil
}

// Start /match/start check [ true true true ] access [ true false false false POST ]
func (a *Actions) Start(s abstract.IState, input game_inputs_match.StartInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	var toolbox = abstract.UseToolbox[*statesl3.ToolboxL3](a.Layer.Tools())
	queueName := input.GameKey + "-" + "friendly"
	q, ok := queues.Get(queueName)
	if !ok {
		return nil, errors.New("game or level not found")
	}
	trx := state.Trx()

	meta := game_model.Meta{Id: input.GameKey}
	trx.Db().First(&meta)
	trx.ClearError()
	if vRaw, ok := meta.Data["maintenance"]; ok {
		if v, ok2 := vRaw.(string); ok2 && (v == "true") {
			return nil, errors.New("in maintenance")
		}
	}
	memberCount := int64(0)
	trx.Db().Model(&model.Member{}).Where("space_id = ?", state.Info().SpaceId()).Count(&memberCount)
	if q.HumanCount != int(memberCount) {
		return nil, errors.New("humans count in the space must be " + fmt.Sprintf("%d", q.HumanCount))
	}
	adminMember := model.Member{}
	trx.Db().Model(&model.Member{}).Where("user_id = ?", state.Info().UserId()).Where("space_id = ?", state.Info().SpaceId()).First(&adminMember)
	trx.ClearError()
	space := model.Space{Id: state.Info().SpaceId()}
	trx.Db().First(&space)
	trx.ClearError()
	space.Tag = "match_friendly"
	trx.Db().Save(&space)
	trx.ClearError()
	topic := model.Topic{Id: state.Info().TopicId()}
	trx.Db().First(&topic)
	trx.ClearError()

	neededBotCount := (1 + q.Bot.Friends + q.Bot.Oponents) - int(memberCount)

	var members = map[string]model.Member{}
	humanMems := []model.Member{}
	trx.Db().Model(&model.Member{}).Where("space_id = ?", state.Info().SpaceId()).Find(&humanMems)
	trx.ClearError()
	players := []string{}
	for _, hm := range humanMems {
		players = append(players, hm.UserId)
		members[hm.Id] = hm
	}
	for i := 0; i < neededBotCount; i++ {
		players = append(players, q.Bot.UserId)
		var member model.Member
		ti := topic.Id
		if ti == "" {
			ti = "*"
		}
		member = model.Member{Id: crypto.SecureUniqueId(a.Layer.Core().Id()), UserId: q.Bot.UserId, SpaceId: space.Id, TopicId: ti, Metadata: ""}
		err2 := trx.Db().Create(&member).Error
		if err2 != nil {
			log.Println(err2)
		}
		trx.ClearError()
		trx.Mem().Put(fmt.Sprintf(memberTemplate, member.SpaceId, member.UserId, member.Id), member.TopicId)
		toolbox.Signaler().JoinGroup(member.SpaceId, member.UserId)
		members[member.Id] = member
	}
	callback := startGame(a, true, toolbox.Signaler(), trx, q, toolbox, adminMember.Id, space, topic, input.GameKey, players, members)
	return callback, nil
}

func handleEffect(a *Actions, gameKey string, level string, userId string, state states.IStateL1, meta game_model.Meta, gameResKey string) error {
	queueName := gameKey + "-" + level
	gameDataStr := ""
	trx := state.Trx()
	trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", gameKey)).Where("id = ?", userId).First(&gameDataStr)
	trx.ClearError()
	oldVals := map[string]interface{}{}
	err2 := json.Unmarshal([]byte(gameDataStr), &oldVals)
	if err2 != nil {
		log.Println(err2)
		return err2
	}
	oldVal := oldVals["games"].(float64)
	err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", userId) }, &model.User{Id: userId}, "metadata", gameKey+".games", 1+oldVal)
	if err != nil {
		log.Println(err)
	}
	trx.ClearError()
	if gameResKey == "win" {
		oldVal := oldVals["wins"].(float64)
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", userId) }, &model.User{Id: userId}, "metadata", gameKey+".wins", 1+oldVal)
		if err != nil {
			log.Println(err)
		}
		trx.ClearError()
	}
	q, _ := queues.Get(queueName)
	for k, v := range q.Effects {
		delta := meta.Data["room"+level+gameResKey+v].(float64)
		oldVal := oldVals[k].(float64)
		res := delta + oldVal
		if res < 0 {
			res = 0
		}
		err := adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", userId) }, &model.User{Id: userId}, "metadata", gameKey+"."+k, res)
		if err != nil {
			log.Println(err)
			return err
		}
		trx.ClearError()
		targetLB, ok := q.LeaderBoard[v]
		if ok {
			if targetLB.Sharded {
				oldShardIndex := 0
				newShardIndex := 0
				for i := len(targetLB.Connectors) - 1; i >= 0; i-- {
					if targetLB.Connectors[i].Start <= oldVal {
						oldShardIndex = i
						break
					}
				}
				for i := len(targetLB.Connectors) - 1; i >= 0; i-- {
					if targetLB.Connectors[i].Start <= res {
						newShardIndex = i
						break
					}
				}
				if res == 0 {
					lb := targetLB.Connectors[oldShardIndex]
					newState := a.Layer.Sb().NewState(module_actor_model.NewGodInfo(userId, "", "", true, ""), trx)
					_, _, err := a.Layer.Core().Get(1).Actor().FetchAction("/admin/board/kickout").Act(newState, admin_inputs_board.KickoutInput{GameKey: gameKey, Level: lb.LBLevel, UserId: userId})
					if err != nil {
						log.Println(err)
					}
					trx.ClearError()
				} else if newShardIndex == oldShardIndex {
					lb := targetLB.Connectors[newShardIndex]
					data := map[string]any{}
					data[lb.Key] = delta
					newState2 := a.Layer.Sb().NewState(module_actor_model.NewInfo(userId, "", "", ""), trx)
					_, _, err2 := a.Layer.Actor().FetchAction("/board/submit").Act(newState2, game_inputs_board.SubmitInput{GameKey: gameKey, Level: lb.LBLevel, Data: data})
					if err2 != nil {
						log.Println(err2)
					}
					trx.ClearError()
				} else {
					lb := targetLB.Connectors[oldShardIndex]
					newState := a.Layer.Sb().NewState(module_actor_model.NewGodInfo(userId, "", "", true, ""), trx)
					_, _, err := a.Layer.Core().Get(1).Actor().FetchAction("/admin/board/kickout").Act(newState, admin_inputs_board.KickoutInput{GameKey: gameKey, Level: lb.LBLevel, UserId: userId})
					if err != nil {
						log.Println(err)
					}
					trx.ClearError()
					lb = targetLB.Connectors[newShardIndex]
					data := map[string]any{}
					data[lb.Key] = res
					newState2 := a.Layer.Sb().NewState(module_actor_model.NewInfo(userId, "", "", ""), trx)
					_, _, err2 := a.Layer.Actor().FetchAction("/board/submit").Act(newState2, game_inputs_board.SubmitInput{GameKey: gameKey, Level: lb.LBLevel, Data: data})
					if err2 != nil {
						log.Println(err2)
					}
					trx.ClearError()
				}
			} else {
				if res == 0 {
					for _, lb := range targetLB.Connectors {
						newState := a.Layer.Sb().NewState(module_actor_model.NewGodInfo(userId, "", "", true, ""), trx)
						_, _, err := a.Layer.Core().Get(1).Actor().FetchAction("/admin/board/kickout").Act(newState, admin_inputs_board.KickoutInput{GameKey: gameKey, Level: lb.LBLevel, UserId: userId})
						if err != nil {
							log.Println(err)
						}
						trx.ClearError()
					}
				} else {
					for _, lb := range targetLB.Connectors {
						data := map[string]any{}
						data[lb.Key] = delta
						newState := a.Layer.Sb().NewState(module_actor_model.NewInfo(userId, "", "", ""), trx)
						_, _, err := a.Layer.Actor().FetchAction("/board/submit").Act(newState, game_inputs_board.SubmitInput{GameKey: gameKey, Level: lb.LBLevel, Data: data})
						if err != nil {
							log.Println(err)
						}
						trx.ClearError()
					}
				}
			}
		}
	}
	return nil
}

// PostStart /match/postStart check [ true false false ] access [ true false false false POST ]
func (a *Actions) PostStart(s abstract.IState, input game_inputs_match.PostStartInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	q, ok := queues.Get(input.GameKey + "-" + input.Level)
	if !ok || (state.Info().UserId() != q.GodId) {
		return nil, errors.New("access denied")
	}

	meta := game_model.Meta{Id: input.GameKey}
	err := trx.Db().First(&meta).Error
	if err != nil {
		return nil, err
	}

	fee := meta.Data["room"+input.Level+"fee"].(float64)

	for _, player := range input.Humans {
		if (player == q.GodId) || (player == q.Bot.UserId) {
			continue
		}
		gameDataStr := ""
		trx.Db().Model(&model.User{}).Select(adapters.BuildJsonFetcher("metadata", input.GameKey)).Where("id = ?", player).First(&gameDataStr)
		trx.ClearError()
		oldVals := map[string]interface{}{}
		err2 := json.Unmarshal([]byte(gameDataStr), &oldVals)
		if err2 != nil {
			log.Println(err2)
		}
		oldVal := oldVals[q.Fee].(float64)
		newVal := oldVal - fee
		if newVal < 0 {
			newVal = 0
		}
		adapters.UpdateJson(func() *gorm.DB { return trx.Db().Model(&model.User{}).Where("id = ?", player) }, &model.User{Id: player}, "metadata", input.GameKey+"."+q.Fee, newVal)
		trx.ClearError()
	}
	return game_outputs_match.EndOutput{}, nil
}

// End /match/end check [ true false false ] access [ true false false false POST ]
func (a *Actions) End(s abstract.IState, input game_inputs_match.EndInput) (any, error) {
	var state = abstract.UseState[states.IStateL1](s)
	trx := state.Trx()

	q, ok := queues.Get(input.GameKey + "-" + input.Level)
	if !ok || (state.Info().UserId() != q.GodId) {
		return nil, errors.New("access denied")
	}

	meta := game_model.Meta{Id: input.GameKey}
	err := trx.Db().First(&meta).Error
	if err != nil {
		return nil, err
	}

	for _, winner := range input.Winners {
		if (winner == q.Bot.UserId) || (strings.HasPrefix(winner, "b_")) {
			continue
		}
		err := handleEffect(a, input.GameKey, input.Level, winner, state, meta, "win")
		if err != nil {
			return nil, err
		}
	}
	for _, looser := range input.Loosers {
		if (looser == q.Bot.UserId) || (strings.HasPrefix(looser, "b_")) {
			continue
		}
		err := handleEffect(a, input.GameKey, input.Level, looser, state, meta, "lose")
		if err != nil {
			return nil, err
		}
	}

	return game_outputs_match.EndOutput{}, nil
}
