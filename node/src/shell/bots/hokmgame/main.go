package hokmgame

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kasper/src/abstract"
	inputs_topics "kasper/src/shell/api/inputs/topics"
	"kasper/src/shell/bots/hokmgame/inputs"
	"kasper/src/shell/bots/hokmgame/models"
	"kasper/src/shell/layer1/adapters"
	"kasper/src/shell/utils/future"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	module_model "kasper/src/shell/layer2/model"

	cmap "github.com/orcaman/concurrent-map/v2"
)

var mapOfNameToIndex = map[string]int{
	"2":  0,
	"3":  1,
	"4":  2,
	"5":  3,
	"6":  4,
	"7":  5,
	"8":  6,
	"9":  7,
	"10": 8,
	"11": 9,  //"sarbaz",
	"12": 10, //"bibi",
	"13": 11, //"shah",
	"14": 12, //"as",
}

var cardTypes = map[string]bool{
	"d": true,
	"p": true,
	"g": true,
	"k": true,
}

const playerCount = 4

// types ----------------------------------------------------------

type PlayerState struct {
	MissCount    int
	NotAvailable bool
	Ready        bool
	Ready2       bool
	TextsCount   int
	EmojiCount   int
}

type Game struct {
	controller          *HokmGame               `json:"-"`
	Started             bool                    `json:"started"`
	IsFriendly          bool                    `json:"isFriendly"`
	TimeoutMemberId     string                  `json:"timeoutMemberId"`
	Level               string                  `json:"level"`
	Turns               float64                 `json:"turns"`
	Manager             map[string]*PlayerState `json:"manager"`
	SpaceId             string                  `json:"spaceId"`
	TopicId             string                  `json:"topicId"`
	MemberId            string                  `json:"memberId"`
	MembersToPlayersMap map[string]string       `json:"membersToPlayersMap"`
	MembersToTeamsMap   map[string]string       `json:"membersToTeamsMap"`
	Teams               map[string]*models.Team `json:"teams"`
	Players             []models.Player         `json:"players"`
	MembersIndex        map[string]int          `json:"membersIndex"`
	Hakem               string                  `json:"hakem"`
	Hokm                string                  `json:"hokm"`
	MapOfOptions        map[string]string       `json:"mapOfOptions"`
	History             []*GameSet              `json:"history"`
	StageCard           string                  `json:"stageCard"`
	PrevWinner          string                  `json:"prevWinner"`
	HakemReady          bool                    `json:"hakemReady"`
	UserReady           bool                    `json:"userReady"`
	Round               int                     `json:"round"`
	CardGroups          map[string][]string     `json:"cardGroups"`
}

type GameSet struct {
	Options []string `json:"options"`
}

type GameRecord struct {
	Id   string `json:"id" gorm:"primaryKey;column:id"`
	Data string `json:"data" gorm:"column:data"`
}

type HokmGame struct {
	Core  abstract.ICore
	Store adapters.IStorage
}

var Queues *cmap.ConcurrentMap[string, chan func()]
var heartbeat = time.NewTicker(time.Duration(5) * time.Second)

func (h *HokmGame) Install(c abstract.ICore, t string) {
	token = t
	h.Core = c
	h.Store = abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools()).Storage()
	h.Store.AutoMigrate(&GameRecord{})
	qs := cmap.New[chan func()]()
	Queues = &qs
}

// functions ---------------------------------------------------------

func (h *HokmGame) createGame(level string, turns float64, spaceId string, topicId string, memberId string, ps []models.Player, isFriendly bool) *Game {
	game := Game{
		Started:             false,
		IsFriendly:          isFriendly,
		Level:               level,
		Turns:               turns,
		SpaceId:             spaceId,
		TopicId:             topicId,
		MemberId:            memberId,
		MembersToPlayersMap: make(map[string]string),
		MembersToTeamsMap:   make(map[string]string),
		Teams:               make(map[string]*models.Team),
		Players:             make([]models.Player, len(ps)),
		MembersIndex:        make(map[string]int),
		Hakem:               "",
		Hokm:                "",
		MapOfOptions:        make(map[string]string),
		History:             make([]*GameSet, 0),
		StageCard:           "",
		PrevWinner:          "",
		HakemReady:          false,
		UserReady:           false,
		Round:               0,
		Manager:             map[string]*PlayerState{},
		CardGroups:          map[string][]string{},
		controller:          h,
	}
	game.Players = ps
	for i, p := range ps {
		game.Manager[p.MemberId] = &PlayerState{MissCount: 0, NotAvailable: false, Ready: false, Ready2: false, TextsCount: 0, EmojiCount: 0}
		game.MembersToPlayersMap[p.MemberId] = p.UserId
		game.MembersIndex[p.MemberId] = i
		game.MembersToTeamsMap[p.MemberId] = p.TeamId
		_, ok := game.Teams[p.TeamId]
		if !ok {
			game.Teams[p.TeamId] = &models.Team{Id: p.TeamId, Score: 0, MainScore: 0}
		}
	}

	fmt.Println("choosing hakem...")
	game.makeHakem()
	fmt.Println("shuffling cards...")
	game.makeCardSet()
	fmt.Println("starting first set...")
	game.startNewSet()

	gameStr, err := json.Marshal(game)
	if err != nil {
		panic(err)
	}
	h.Store.Db().Create(GameRecord{Id: topicId, Data: string(gameStr)})
	if Queues.SetIfAbsent(topicId, make(chan func(), 1)) {
		future.Async(func() {
			for {
				q, ok := Queues.Get(topicId)
				if !ok {
					break
				}
				select {
				case m := <-q:
					m()
				case <-heartbeat.C:
				}
			}
		}, false)
	}

	return &game
}

func (g *Game) makeHakem() {
	var num = rand.Intn(playerCount)
	g.Hakem = g.Players[num].MemberId
	g.PrevWinner = g.Hakem
}

func (g *Game) makeHokm(user string, cardType string) (bool, error) {
	if g.Hakem == user {
		if cardTypes[cardType] {
			g.Hokm = cardType
			return true, nil
		} else {
			return false, errors.New("invalid card type")
		}
	} else {
		return false, errors.New("you are not hakem")
	}
}

func (g *Game) makeCardSet() {
	slice := []string{}
	for key := range cardTypes {
		for j := 0; j < len(mapOfNameToIndex); j++ {
			option := fmt.Sprintf("%s-%d", key, j+2)
			slice = append(slice, option)
		}
	}
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
	eachUserCardCount := (len(cardTypes) * len(mapOfNameToIndex) / playerCount)
	g.MapOfOptions = map[string]string{}
	for i := 0; i < playerCount; i++ {
		for j := 0; j < eachUserCardCount; j++ {
			g.MapOfOptions[slice[i*eachUserCardCount+j]] = g.Players[i].MemberId
		}
	}
}

func (g *Game) startNewSet() {
	g.History = append(g.History, &GameSet{Options: []string{}})
}

func (g *Game) notfiyWinnerToServer(winnerTeam string) {
	if g.IsFriendly {
		return
	}
	winners := []string{}
	loosers := []string{}
	for _, player := range g.Players {
		if g.Manager[player.MemberId].NotAvailable {
			continue
		}
		if player.TeamId == winnerTeam {
			winners = append(winners, player.UserId)
		} else {
			loosers = append(loosers, player.UserId)
		}
	}
	data := map[string]any{
		"gameKey": "hokm",
		"level":   g.Level,
		"winners": winners,
		"loosers": loosers,
	}

	future.Async(func() { doHttpCall("/match/end", "3", data) }, false)
}

func (g *Game) notfiyStartToServer() {
	if g.Started {
		return
	}
	if g.IsFriendly {
		return
	}
	g.Started = true
	humanPlayers := []string{}
	for _, player := range g.Players {
		if strings.HasPrefix(player.UserId, "b_") {
			continue
		}
		humanPlayers = append(humanPlayers, player.UserId)
	}

	data := map[string]any{
		"gameKey": "hokm",
		"level":   g.Level,
		"humans":  humanPlayers,
	}
	g.controller.SaveGame(g)

	future.Async(func() { doHttpCall("/match/postStart", "3", data) }, false)
}

func (g *Game) act(userMemberId string, option string) (bool, int, string, func(), string, error) {
	if !strings.Contains(option, "-") {
		return false, 0, "", nil, "", errors.New("action is invalid")
	}
	if len(g.History[len(g.History)-1].Options) == 0 {
		if userMemberId != g.PrevWinner {
			return false, 0, "", nil, "", errors.New("you are not starter")
		}
	}
	var pos = (g.MembersIndex[userMemberId] + playerCount - g.MembersIndex[g.PrevWinner]) % playerCount
	if pos != (len(g.History[len(g.History)-1].Options)) {
		return false, 0, "", nil, "", errors.New("not your turn")
	}
	if (len(g.History[len(g.History)-1].Options) > 0) && (strings.Split(option, "-")[0] != g.StageCard) {
		for key := range g.MapOfOptions {
			if (g.MapOfOptions[key] == userMemberId) && (strings.Split(key, "-")[0] == g.StageCard) {
				return false, 0, "", nil, "", errors.New("action not available while you have stage type cards")
			}
		}
	}
	if g.MapOfOptions[option] != userMemberId {
		return false, 0, "", nil, "", errors.New("option not available")
	}
	if len(g.History[len(g.History)-1].Options) == 0 {
		g.StageCard = strings.Split(option, "-")[0]
	}
	if strings.Split(option, "-")[0] != g.StageCard {
		for key, val := range g.MapOfOptions {
			if val == userMemberId {
				if strings.Split(key, "-")[0] == g.StageCard {
					return false, 0, "", nil, "", errors.New("you cant use non-stage card while you have stage card")
				}
			}
		}
	}
	g.MapOfOptions[option] = ""
	g.History[len(g.History)-1].Options = append(g.History[len(g.History)-1].Options, option)
	if len(g.History[len(g.History)-1].Options) == playerCount {
		winner := g.checkWinner()
		g.PrevWinner = winner
		g.Teams[g.MembersToTeamsMap[winner]].Score++
		teamsList := []*models.Team{}
		for _, team := range g.Teams {
			teamsList = append(teamsList, team)
		}
		var hakemTeam = g.Teams[g.MembersToTeamsMap[g.Hakem]].Id
		stageEnded := false
		var winnerTeam = ""
		if (teamsList[0].Score >= 7) && (teamsList[1].Score == 0) {
			teamsList[0].MainScore += 2
			winnerTeam = teamsList[0].Id
			stageEnded = true
			if hakemTeam != winnerTeam {
				teamsList[0].MainScore++
			}
		} else if (teamsList[1].Score >= 7) && (teamsList[0].Score == 0) {
			teamsList[1].MainScore += 2
			winnerTeam = teamsList[1].Id
			stageEnded = true
			if hakemTeam != winnerTeam {
				teamsList[1].MainScore++
			}
		} else if teamsList[0].Score >= 7 {
			teamsList[0].MainScore++
			winnerTeam = teamsList[0].Id
			stageEnded = true
		} else if teamsList[1].Score >= 7 {
			teamsList[1].MainScore++
			winnerTeam = teamsList[1].Id
			stageEnded = true
		}
		if stageEnded {
			g.Round = 0
			stepScores := map[string]float64{}
			stepScores[teamsList[0].Id] = teamsList[0].Score
			stepScores[teamsList[1].Id] = teamsList[1].Score
			if hakemTeam != winnerTeam {
				hakemIndex := g.MembersIndex[g.Hakem]
				newHakemIndex := hakemIndex + 1
				if newHakemIndex > (playerCount - 1) {
					newHakemIndex = 0
				}
				g.Hakem = g.Players[newHakemIndex].MemberId
			}
			g.PrevWinner = g.Hakem
			g.Hokm = ""
			teamsList[0].Score = 0
			teamsList[1].Score = 0
			if (teamsList[0].MainScore >= g.Turns) || (teamsList[1].MainScore >= g.Turns) {
				scores := map[string]float64{}
				scores[teamsList[0].Id] = teamsList[0].MainScore
				scores[teamsList[1].Id] = teamsList[1].MainScore
				setTimeout(func() {
					g.notifyDestruction()
					g.controller.Store.Db().Delete(&GameRecord{Id: g.TopicId})
					Queues.Remove(g.TopicId)
				}, 60, g.TopicId)
				if teamsList[0].MainScore > teamsList[1].MainScore {
					return true, 3, "", func() {
						g.notifyStageResult(winner, winnerTeam)
						g.notifyStepResult(winnerTeam, stepScores)
						g.notfiyWinnerToServer(teamsList[0].Id)
						g.notifyGameResult(teamsList[0].Id, scores)
					}, winnerTeam, nil
				} else {
					return true, 3, "", func() {
						g.notifyStageResult(winner, winnerTeam)
						g.notifyStepResult(winnerTeam, stepScores)
						g.notfiyWinnerToServer(teamsList[1].Id)
						g.notifyGameResult(teamsList[1].Id, scores)
					}, winnerTeam, nil
				}
			}
			return true, 2, g.Hakem, func() {
				g.notifyStageResult(winner, winnerTeam)
				g.notifyStepResult(winnerTeam, stepScores)
			}, winnerTeam, nil
		}
		g.Round++
		g.startNewSet()
		wt := ""
		for _, p := range g.Players {
			if p.MemberId == winner {
				wt = p.TeamId
				break
			}
		}
		return true, 1, winner, func() {
			g.notifyStageResult(winner, wt)
		}, wt, nil
	}
	return true, 1, g.Players[(g.MembersIndex[g.PrevWinner]+pos+1)%playerCount].MemberId, nil, "", nil
}

func (g *Game) checkWinner() string {
	options := g.History[len(g.History)-1].Options
	mostValuable := ""
	mostValuableUser := ""
	prevWinnerIndex := g.MembersIndex[g.PrevWinner]
	type extraRuler struct {
		memberId string
		opt      string
	}
	extraRule := []extraRuler{}
	for i, opt := range options {
		if mostValuableUser == "" {
			mostValuable = opt
			mostValuableUser = g.Players[(prevWinnerIndex+i)%playerCount].MemberId
			continue
		}
		if strings.Split(opt, "-")[0] != g.StageCard {
			if strings.Split(opt, "-")[0] == g.Hokm {
				extraRule = append(extraRule, extraRuler{memberId: g.Players[(prevWinnerIndex+i)%playerCount].MemberId, opt: opt})
			} else {
				continue
			}
		}
		if mapOfNameToIndex[strings.Split(mostValuable, "-")[1]] < mapOfNameToIndex[strings.Split(opt, "-")[1]] {
			mostValuable = opt
			mostValuableUser = g.Players[(prevWinnerIndex+i)%playerCount].MemberId
		}
	}
	if len(extraRule) == 0 {
		return mostValuableUser
	} else {
		winner := ""
		opt := ""
		for _, ruler := range extraRule {
			if (opt == "") || (mapOfNameToIndex[strings.Split(opt, "-")[1]] < mapOfNameToIndex[strings.Split(ruler.opt, "-")[1]]) {
				winner = ruler.memberId
				opt = ruler.opt
			}
		}
		return winner
	}
}

func (g *Game) notifyDestruction() {
	g.controller.SendTopicPacket("broadcast", g.SpaceId, g.TopicId, g.MemberId, "",
		map[string]any{
			"type": "destruct",
		})
}

func (g *Game) notifyGameResult(winner string, scores map[string]float64) {
	g.controller.SendTopicPacket("broadcast", g.SpaceId, g.TopicId, g.MemberId, "",
		map[string]any{
			"type":       "gameResult",
			"winner":     winner,
			"gameScores": scores,
		})
}

func (g *Game) notifyStepResult(winner string, scores map[string]float64) {
	for _, p := range g.Players {
		g.controller.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, p.MemberId,
			map[string]any{
				"type":       "stepResult",
				"winner":     winner,
				"stepScores": scores,
			})
	}
}

func (g *Game) notifyStageResult(winner string, winnerTeam string) {
	for _, p := range g.Players {
		g.controller.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, p.MemberId,
			map[string]any{
				"type":       "stageResult",
				"winner":     winner,
				"winnerTeam": winnerTeam,
			})
	}
}

func (h *HokmGame) notifyGameCreation(g *Game) {
	type exportedGame struct {
		Players []models.Player `json:"players"`
		Hakem   string          `json:"hakem"`
		Cards   []string        `json:"cards"`
	}
	var hakemIsHuman = false
	for _, player := range g.Players {
		if !strings.HasPrefix(player.UserId, "b_") && g.Hakem == player.MemberId {
			hakemIsHuman = true
			break
		}
	}
	randN := 0
	if hakemIsHuman {
		arr := []int{5, 9}
		randN = arr[rand.Intn(len(arr))]
	} else {
		arr := []int{1, 5, 9}
		randN = arr[rand.Intn(len(arr))]
	}
	for _, player := range g.Players {
		eg := exportedGame{Hakem: g.Hakem, Players: g.Players, Cards: []string{}}
		g.CardGroups[player.MemberId] = []string{}
		for card, owner := range g.MapOfOptions {
			if owner == player.MemberId {
				g.CardGroups[player.MemberId] = append(g.CardGroups[player.MemberId], card)
				eg.Cards = append(eg.Cards, card)
			}
		}
		type data struct {
			Type     string       `json:"type"`
			Game     exportedGame `json:"game"`
			TeamId   string       `json:"teamId"`
			RndNum   int          `json:"rndNum"`
			NextTurn string       `json:"nextTurn"`
		}
		rn := 0
		if !strings.HasPrefix(player.UserId, "b_") {
			rn = randN
		}
		var dataObj = data{
			Type:   "gameCreation",
			Game:   eg,
			TeamId: player.TeamId,
			RndNum: rn,
		}
		h.SaveGame(g)
		h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, player.MemberId, dataObj)
	}
}

func (h *HokmGame) askHokm(g *Game) {
	for _, player := range g.Players {
		if player.MemberId == g.Hakem {
			type data struct {
				Type string `json:"type"`
			}
			var dataObj = data{
				Type: "tellMeHokm",
			}
			h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, player.MemberId, dataObj)
			break
		}
	}
}

func (h *HokmGame) notifyGameStart(g *Game, starter string) {
	currentStep := len(g.History)
	currentIndex := len(g.History[len(g.History)-1].Options)
	if g.Manager[starter].NotAvailable {
		g.TimeoutMemberId = starter
		h.SaveGame(g)
		robotId := ""
		for _, player := range g.Players {
			if strings.HasPrefix(player.UserId, "b_") {
				robotId = player.MemberId
				break
			}
		}
		opts := []string{}
		for k, v := range g.MapOfOptions {
			if v == starter {
				opts = append(opts, k)
			}
		}
		setTimeout(func() {
			h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, robotId,
				map[string]any{
					"type":          "simGame",
					"humanPlayerId": starter,
					"options":       opts,
				})
		}, int64(rand.Intn(3)+3), g.TopicId)
		return
	}
	topicId := g.TopicId
	setTimeout(func() {
		g := h.GetGame(topicId)
		if g == nil {
			return
		}
		currentStep2 := len(g.History)
		currentIndex2 := len(g.History[len(g.History)-1].Options)
		if (currentStep == currentStep2) && (currentIndex == currentIndex2) {
			g.TimeoutMemberId = starter
			g.Manager[starter].MissCount++
			h.SaveGame(g)
			if g.Manager[starter].MissCount > 1 {
				if !g.Manager[starter].NotAvailable {
					g.Manager[starter].NotAvailable = true
					h.SaveGame(g)
					h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, starter,
						map[string]any{
							"type": "kicked",
						})
					if !g.IsFriendly {
						winners := []string{}
						loosers := []string{}
						for _, player := range g.Players {
							if player.MemberId == starter {
								loosers = append(loosers, player.UserId)
								break
							}
						}
						data := map[string]any{
							"gameKey": "hokm",
							"level":   g.Level,
							"winners": winners,
							"loosers": loosers,
						}

						future.Async(func() { doHttpCall("/match/end", "3", data) }, false)
					}
				}
				dead := true
				for i := 0; i < len(g.Players); i++ {
					if !strings.HasPrefix(g.Players[i].UserId, "b_") && !g.Manager[g.Players[i].MemberId].NotAvailable {
						dead = false
						break
					}
				}
				if dead {
					g.notifyDestruction()
					g.controller.Store.Db().Delete(&GameRecord{Id: g.TopicId})
					Queues.Remove(g.TopicId)
					return
				}
			}
			robotId := ""
			for _, player := range g.Players {
				if strings.HasPrefix(player.UserId, "b_") {
					robotId = player.MemberId
					break
				}
			}
			opts := []string{}
			for k, v := range g.MapOfOptions {
				if v == starter {
					opts = append(opts, k)
				}
			}
			h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, robotId,
				map[string]any{
					"type":          "simGame",
					"humanPlayerId": starter,
					"options":       opts,
				})
		}
	}, 15, topicId)
	if g.TimeoutMemberId == "" {
		h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, starter,
			map[string]any{
				"type": "startGame",
			})
	}
}

func (h *HokmGame) notifyHokmSpecification(g *Game, hokm string, fromGod bool) {
	for _, player := range g.Players {
		h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, player.MemberId,
			map[string]any{
				"type":    "hokmSpecification",
				"hokm":    hokm,
				"fromGod": fromGod,
			})
	}
}

func (h *HokmGame) notifyGamePlay(g *Game, playerMemberId, action string, round int, fromGod bool) {
	for _, player := range g.Players {
		h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, player.MemberId,
			map[string]any{
				"type":           "gamePlay",
				"playerMemberId": playerMemberId,
				"action":         action,
				"round":          round,
				"fromGod":        fromGod,
			})
	}
}

func OnAddToGroup(input any) any {

	_, err := json.Marshal(input)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{}
}

func extractData[T any](data any) T {
	inp := new(T)
	str, err2 := json.Marshal(data)
	if err2 != nil {
		log.Println(err2.Error())
		log.Println("wrong input structure")
	}
	err3 := json.Unmarshal(str, inp)
	if err3 != nil {
		log.Println(err3.Error())
		log.Println("wrong input structure")
	}
	return *inp
}

func (h *HokmGame) suggestReact(g *Game, memberId string, positive bool, ofHokm bool) {
	if strings.HasPrefix(g.MembersToPlayersMap[memberId], "b_") {
		if rand.Intn(4) > 0 {
			return
		}
		positiveTextPool := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		negativeTextPool := []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		positiveEmojiPool := []int{0, 1, 4, 10}
		negativeEmojiPool := []int{2, 3, 5, 6, 7}
		textOrEmoji := rand.Intn(2)
		if textOrEmoji == 0 {
			if g.Manager[memberId].TextsCount > 2 {
				if g.Manager[memberId].EmojiCount > 2 {
					textOrEmoji = -1
				} else {
					textOrEmoji = 1
					g.Manager[memberId].EmojiCount++
				}
			} else {
				g.Manager[memberId].TextsCount++
			}
		} else if textOrEmoji == 1 {
			if g.Manager[memberId].EmojiCount > 2 {
				if g.Manager[memberId].TextsCount > 2 {
					textOrEmoji = -1
				} else {
					textOrEmoji = 0
					g.Manager[memberId].TextsCount++
				}
			} else {
				g.Manager[memberId].EmojiCount++
			}
		}
		if textOrEmoji == -1 {
			return
		}
		var choice int
		var choiceType string
		if textOrEmoji == 0 {
			if positive {
				choice = positiveTextPool[rand.Intn(len(positiveTextPool))]
				choiceType = "text"
			} else {
				choice = negativeTextPool[rand.Intn(len(negativeTextPool))]
				choiceType = "text"
			}
		} else {
			if positive {
				choice = positiveEmojiPool[rand.Intn(len(positiveEmojiPool))]
				choiceType = "emoji"
			} else {
				choice = negativeEmojiPool[rand.Intn(len(negativeEmojiPool))]
				choiceType = "emoji"
			}
		}
		h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, memberId,
			map[string]any{
				"type":       "suggestReact",
				"choice":     fmt.Sprintf("%d", choice),
				"choiceType": choiceType,
			})
	} else {
		if g.Manager[memberId].TextsCount < 4 {
			var r int
			if ofHokm {
				r = rand.Intn(5)
			} else {
				r = rand.Intn(4)
			}
			if r == 0 {
				positiveTextPool := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
				negativeTextPool := []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
				var choices []int
				if positive {
					rand.Shuffle(len(positiveTextPool), func(i, j int) {
						positiveTextPool[i], positiveTextPool[j] = positiveTextPool[j], positiveTextPool[i]
					})
					choices = positiveTextPool[0:3]
				} else {
					rand.Shuffle(len(negativeTextPool), func(i, j int) {
						negativeTextPool[i], negativeTextPool[j] = negativeTextPool[j], negativeTextPool[i]
					})
					choices = negativeTextPool[0:3]
				}
				h.SendTopicPacket("single", g.SpaceId, g.TopicId, g.MemberId, memberId,
					map[string]any{
						"type":  "textOptions",
						"texts": choices,
					})
				g.Manager[memberId].TextsCount++
			}
		}
	}
}

func (h *HokmGame) SaveGame(g *Game) {
	gameStr, err := json.Marshal(g)
	if err != nil {
		log.Println(err)
		return
	}
	err2 := h.Store.Db().Save(&GameRecord{Id: g.TopicId, Data: string(gameStr)}).Error
	if err2 != nil {
		log.Println(err2)
	}
}

func (h *HokmGame) GetGame(topicId string) *Game {
	g := new(Game)
	gameRecord := GameRecord{Id: topicId}
	e := h.Store.Db().First(&gameRecord).Error
	if e != nil {
		log.Println(e)
		return nil
	}
	errM := json.Unmarshal([]byte(gameRecord.Data), g)
	if errM != nil {
		log.Println(errM)
		return nil
	}
	g.controller = h
	return g
}

func (h *HokmGame) waitForHakemReady(topicId string) {
	setTimeout(func() {
		g := h.GetGame(topicId)
		if g == nil {
			return
		}
		if !g.HakemReady {
			g.HakemReady = true
			h.SaveGame(g)
			if g.HakemReady {
				g.notfiyStartToServer()
				setTimeout(func() {
					g := h.GetGame(topicId)
					if g == nil {
						return
					}
					if g.Hokm == "" {
						rndIndex := rand.Intn(len(cardTypes))
						var cts = []string{
							"d",
							"p",
							"g",
							"k",
						}
						hokm := cts[rndIndex]
						g.makeHokm(g.Hakem, hokm)
						h.SaveGame(g)
						h.notifyHokmSpecification(g, hokm, true)
						setTimeout(func() {
							g := h.GetGame(topicId)
							if g == nil {
								return
							}
							if !g.UserReady {
								g.UserReady = true
								h.SaveGame(g)
								if g.UserReady {
									h.notifyGameStart(g, g.Hakem)
								}
							}
						}, 15, g.TopicId)
					}
				}, 15, g.TopicId)
				h.askHokm(g)
			}
		}
	}, 15, topicId)
}

func (h *HokmGame) OnTopicSend(input models.Send) any {

	var data = map[string]any{}
	err := json.Unmarshal([]byte(input.Data), &data)
	if err != nil {
		return map[string]any{}
	}

	keyRaw, ok := data["type"]
	if !ok {
		log.Println("no key exist")
		return map[string]any{}
	}
	key, ok2 := keyRaw.(string)
	if !ok2 {
		log.Println("key is not string")
		return map[string]any{}
	}

	task := func() {

		g := new(Game)
		if key != "createGame" {
			ga := h.GetGame(input.Topic.Id)
			if ga == nil {
				return
			}
			g = ga
		}

		switch key {
		case "playGame":
			{
				if g.TimeoutMemberId == input.Member.Id {
					return
				}
				g.Manager[input.Member.Id].MissCount = 0
				act := data["action"].(string)
				currentRound := g.Round
				success, endState, nextTurn, runAfter, winner, err := g.act(input.Member.Id, act)
				h.SaveGame(g)
				if !success {
					log.Println(err.Error())
				} else {
					h.notifyGamePlay(g, input.Member.Id, act, currentRound, false)
					if runAfter != nil {
						runAfter()
					}
					if endState == 2 {
						for _, p := range g.Players {
							if p.TeamId == winner {
								h.suggestReact(g, p.MemberId, true, false)
							} else {
								h.suggestReact(g, p.MemberId, false, false)
							}
						}
						fmt.Println("shuffling cards...")
						g.makeCardSet()
						fmt.Println("starting new step...")
						g.startNewSet()
						g.StageCard = ""
						h.SaveGame(g)
						h.notifyGameCreation(g)
						g.UserReady = false
						g.HakemReady = false
						for _, m := range g.Manager {
							m.Ready2 = false
							m.Ready = false
							m.TextsCount = 0
						}
						h.SaveGame(g)
						h.waitForHakemReady(input.Topic.Id)
					} else if endState == 1 {
						if winner != "" {
							for _, p := range g.Players {
								if p.TeamId == winner {
									h.suggestReact(g, p.MemberId, true, false)
								} else {
									h.suggestReact(g, p.MemberId, false, false)
								}
							}
							h.SaveGame(g)
							setTimeout(func() {
								h.notifyGameStart(g, nextTurn)
							}, int64(rand.Intn(3)+3), g.TopicId)
						} else {
							h.SaveGame(g)
							h.notifyGameStart(g, nextTurn)
						}
					}
					if strings.Split(act, "-")[0] == g.Hokm {
						h.suggestReact(g, input.Member.Id, true, true)
						for i, p := range g.Players {
							if p.MemberId == input.Member.Id {
								if i > 0 {
									h.suggestReact(g, g.Players[i-1].MemberId, false, true)
								}
								break
							}
						}
						h.SaveGame(g)
					}
				}
				break
			}
		case "resSimGame":
			{
				act := data["action"].(string)
				humanPlayerMemberId := data["humanPlayerId"].(string)
				if g.TimeoutMemberId != humanPlayerMemberId {
					return
				}
				g.TimeoutMemberId = ""
				currentRound := g.Round
				success, endState, nextTurn, runAfter, winner, err := g.act(humanPlayerMemberId, act)
				h.SaveGame(g)
				if !success {
					log.Println(err.Error())
				} else {
					h.notifyGamePlay(g, humanPlayerMemberId, act, currentRound, true)
					if runAfter != nil {
						runAfter()
					}
					if endState == 2 {
						for _, p := range g.Players {
							if p.TeamId == winner {
								h.suggestReact(g, p.MemberId, true, false)
							} else {
								h.suggestReact(g, p.MemberId, false, false)
							}
						}
						fmt.Println("shuffling cards...")
						g.makeCardSet()
						fmt.Println("starting new step...")
						g.startNewSet()
						g.StageCard = ""
						h.SaveGame(g)
						h.notifyGameCreation(g)
						g.UserReady = false
						g.HakemReady = false
						for _, m := range g.Manager {
							m.Ready2 = false
							m.Ready = false
							m.TextsCount = 0
						}
						h.SaveGame(g)
						h.waitForHakemReady(input.Topic.Id)
					} else if endState == 1 {
						if winner != "" {
							for _, p := range g.Players {
								if p.TeamId == winner {
									h.suggestReact(g, p.MemberId, true, false)
								} else {
									h.suggestReact(g, p.MemberId, false, false)
								}
							}
							h.SaveGame(g)
							setTimeout(func() {
								h.notifyGameStart(g, nextTurn)
							}, int64(rand.Intn(3)+3), g.TopicId)
						} else {
							h.SaveGame(g)
							h.notifyGameStart(g, nextTurn)
						}
					}
					if strings.Split(act, "-")[0] == g.Hokm {
						for i, p := range g.Players {
							if p.MemberId == input.Member.Id {
								if i > 0 {
									h.suggestReact(g, g.Players[i-1].MemberId, false, true)
								}
								break
							}
						}
						h.SaveGame(g)
					}
				}
				break
			}
		case "leave":
			{
				g.Manager[input.Member.Id].NotAvailable = true
				h.SaveGame(g)
				dead := true
				for i := 0; i < len(g.Players); i++ {
					if !strings.HasPrefix(g.Players[i].UserId, "b_") && !g.Manager[g.Players[i].MemberId].NotAvailable {
						dead = false
						break
					}
				}
				if dead {
					g.notifyDestruction()
					g.controller.Store.Db().Delete(&GameRecord{Id: g.TopicId})
					Queues.Remove(g.TopicId)
				}
				break
			}
		case "createGame":
			{
				inp := extractData[inputs.CreateGameInput](data["value"])
				g = h.createGame(inp.Level, inp.Turns, input.Topic.SpaceId, input.Topic.Id, input.TargetMember.Id, inp.Players, inp.IsFriendly)
				h.notifyGameCreation(g)
				h.waitForHakemReady(input.Topic.Id)
				break
			}
		case "specifyHokm":
			{
				setTimeout(func() {
					g := h.GetGame(input.Topic.Id)
					if g == nil {
						return
					}
					if g.Hokm == "" {
						g.makeHokm(input.Member.Id, data["hokm"].(string))
						h.SaveGame(g)
						h.notifyHokmSpecification(g, data["hokm"].(string), false)
						setTimeout(func() {
							g := h.GetGame(input.Topic.Id)
							if g == nil {
								return
							}
							if !g.UserReady {
								g.UserReady = true
								h.SaveGame(g)
								if g.UserReady {
									h.notifyGameStart(g, g.Hakem)
								}
							}
						}, 15, g.TopicId)
					}
				}, 0, g.TopicId)
				break
			}
		case "hakemReady":
			{
				setTimeout(func() {
					g := h.GetGame(input.Topic.Id)
					if g == nil {
						return
					}
					if !g.HakemReady {
						g.Manager[input.Member.Id].Ready = true
						ready := true
						for memberId, m := range g.Manager {
							if !strings.HasPrefix(g.MembersToPlayersMap[memberId], "b_") {
								if !m.Ready {
									ready = false
									break
								}
							}
						}
						g.HakemReady = ready
						h.SaveGame(g)
						if g.HakemReady {
							g.notfiyStartToServer()
							setTimeout(func() {
								g := h.GetGame(input.Topic.Id)
								if g == nil {
									return
								}
								if g.Hokm == "" {
									var cts = []string{
										"d",
										"p",
										"g",
										"k",
									}
									firstCards := g.CardGroups[g.Hakem][0:5]
									cardCount := []int{0, 0, 0, 0}
									for _, card := range firstCards {
										for i, ct := range cts {
											if ct == strings.Split(card, "-")[0] {
												cardCount[i]++
												break
											}
										}
									}

									maxIndex := 0
									for i := range cts {
										if cardCount[maxIndex] <= cardCount[i] {
											maxIndex = i
										}
									}

									hokm := cts[maxIndex]

									g.makeHokm(g.Hakem, hokm)
									h.SaveGame(g)
									h.notifyHokmSpecification(g, hokm, true)
									setTimeout(func() {
										g := h.GetGame(input.Topic.Id)
										if g == nil {
											return
										}
										if !g.UserReady {
											g.UserReady = true
											h.SaveGame(g)
											if g.UserReady {
												h.notifyGameStart(g, g.Hakem)
											}
										}
									}, 15, g.TopicId)
								}
							}, 15, g.TopicId)
							h.askHokm(g)
						}
					}
				}, 0, g.TopicId)
				break
			}
		case "userReady":
			{
				setTimeout(func() {
					g := h.GetGame(input.Topic.Id)
					if g == nil {
						return
					}
					if !g.UserReady {
						g.Manager[input.Member.Id].Ready2 = true
						ready := true
						for memberId, m := range g.Manager {
							if !strings.HasPrefix(g.MembersToPlayersMap[memberId], "b_") {
								if !m.Ready2 {
									ready = false
									break
								}
							}
						}
						g.UserReady = ready
						h.SaveGame(g)
						if g.UserReady {
							h.notifyGameStart(g, g.Hakem)
						}
					}
				}, 0, g.TopicId)
				break
			}
		}
	}

	if key != "createGame" {
		q, okq := Queues.Get(input.Topic.Id)
		if !okq {
			return map[string]any{}
		}
		q <- task
	} else {
		future.Async(task, false)
	}

	return map[string]any{}
}

type RandGroup struct {
	CallbackId string   `json:"callbackId"`
	Numbers    []string `json:"numbers"`
}

// var address = "185.204.168.179:8080"
// var protocol = ""

var address = "game.midopia.com"
var protocol = "s"

// var address = "localhost:8080"

func doHttpCall(path string, layer string, val any) {

	jsonStr, err := json.Marshal(val)
	if err != nil {
		log.Println(err)
		return
	}
	r, err := http.NewRequest("POST", "http"+protocol+"://"+address+path, bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
		return
	}
	r.Close = true
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("token", token)
	r.Header.Add("layer", layer)

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(result))
	if res.StatusCode != http.StatusOK {
		fmt.Println(errors.New("request failed"))
	}
}

var token = "20aedc96-1771-439d-80d1-3ccbfbaa0581-90a708d5-8cc9-4e7d-aa01-f5fc4a87d2c2"

func setTimeout(callback func(), seconds int64, topicId string) {
	if seconds == 0 {
		q, ok := Queues.Get(topicId)
		if ok {
			q <- callback
		}
	} else {
		future.Async(func() {
			time.Sleep(time.Duration(seconds) * time.Second)
			q, ok := Queues.Get(topicId)
			if ok {
				q <- callback
			}
		}, false)
	}
}

func (h *HokmGame) SendTopicPacket(typ string, spaceId string, topicId string, memberId string, recvId string, data any) {
	innerData, err2 := json.Marshal(data)
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	packet := inputs_topics.SendInput{Type: typ, SpaceId: spaceId, TopicId: topicId, MemberId: memberId, RecvId: recvId, Data: string(innerData)}
	h.Core.Get(1).Actor().FetchAction("/topics/send").(abstract.ISecureAction).SecurelyAct(
		h.Core.Get(1),
		token,
		"",
		packet,
		"",
	)
}
