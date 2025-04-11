package hokmagent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"kasper/src/abstract"
	inputs_topics "kasper/src/shell/api/inputs/topics"
	"kasper/src/shell/bots/hokmagent/models"
	"kasper/src/shell/layer1/adapters"
	module_model "kasper/src/shell/layer2/model"
	"kasper/src/shell/utils/future"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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

var cardTypes = []string{
	"d",
	"p",
	"g",
	"k",
}

// functions ---------------------------------------------------------

type exportedGame struct {
	Players []Player `json:"players"`
	Hakem   string   `json:"hakem"`
	Cards   []string `json:"cards"`
}

type Player struct {
	UserId   string `json:"userId"`
	MemberId string `json:"memberId"`
	TeamId   string `json:"teamId"`
}

type HokmAgent struct {
	Core  abstract.ICore
	Store adapters.IStorage
}

type AgentRecord struct {
	Id      string `json:"id" gorm:"primaryKey;column:id"`
	TopicId string `json:"topicId" gorm:"column:topic_id"`
	Data    string `json:"data" gorm:"column:data"`
}

var heartbeat = time.NewTicker(time.Duration(5) * time.Second)

func (h *HokmAgent) Install(c abstract.ICore, t string) {
	token = t
	h.Core = c
	h.Store = abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools()).Storage()
	h.Store.AutoMigrate(&AgentRecord{})
	qs := cmap.New[chan func()]()
	Queues = &qs
}

func (h *HokmAgent) OnMatchJoin(theMap models.MatchJoinPacket) any {

	gameHolder := &GameHolder{
		ExpGame:        nil,
		Hokm:           "",
		SpaceId:        theMap.Space.Id,
		TopicId:        theMap.Topic.Id,
		GodMemberId:    theMap.GodMember.Id,
		MyMemberId:     theMap.MyMember.Id,
		TurnIndex:      0,
		AvailableCards: map[string]bool{},
		Decks:          []*Deck{{Cards: []string{}}},
		CurrentDeck:    map[string]bool{},
		Simulator:      []string{},
		TmDnHave:       "",
		OpDnHave:       "",
	}

	gameStr, err := json.Marshal(gameHolder)
	if err != nil {
		panic(err)
	}
	h.Store.Db().Create(AgentRecord{Id: theMap.MyMember.Id, TopicId: theMap.Topic.Id, Data: string(gameStr)})

	if Queues.SetIfAbsent(theMap.Topic.Id, make(chan func(), 1)) {
		future.Async(func() {
			for {
				q, ok := Queues.Get(theMap.Topic.Id)
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

	return map[string]any{}
}

func extractData[T any](data any) T {
	inp := new(T)
	str, err2 := json.Marshal(data)
	if err2 != nil {
		log.Println(err2.Error())
		log.Println("wrong input structure")
	}
	err3 := json.Unmarshal(str, &inp)
	if err3 != nil {
		log.Println(err3.Error())
		log.Println("wrong input structure")
	}
	return *inp
}

type Deck struct {
	Cards []string `json:"cards"`
}

type GameHolder struct {
	ExpGame        *exportedGame   `json:"expGame"`
	Hokm           string          `json:"hokm"`
	AvailableCards map[string]bool `json:"availableCards"`
	SpaceId        string          `json:"spaceId"`
	TopicId        string          `json:"topicId"`
	StageType      string          `json:"stageType"`
	GodMemberId    string          `json:"godMemberId"`
	MyMemberId     string          `json:"myMemberId"`
	TurnIndex      int             `json:"turnIndex"`
	Decks          []*Deck         `json:"decks"`
	CurrentDeck    map[string]bool `json:"currentDeck"`
	Simulator      []string        `json:"simulator"`
	Round          float64         `json:"round"`
	EmojiCount     int             `json:"emojiCount"`
	TextCount      int             `json:"textCount"`
	TmDnHave       string          `json:"tmDnHave"`
	OpDnHave       string          `json:"opDnHave"`
	TeamId         string          `json:"teamId"`
}

func (h *HokmAgent) sendPacket(agent *GameHolder, action string) {
	h.SendTopicPacket("single", agent.SpaceId, agent.TopicId, agent.MyMemberId, agent.GodMemberId,
		map[string]any{
			"type":   "playGame",
			"action": action,
		})
}

func getMostUpperValidCard(usedCards map[string]bool, avCards []string, hokm string, tmdhType string, opdnType string) string {
	avCardsArr := avCards[:]
	sort.Slice(avCardsArr, func(i, j int) bool {
		card1 := avCardsArr[i]
		card2 := avCardsArr[j]
		return mapOfNameToIndex[strings.Split(card1, "-")[1]] < mapOfNameToIndex[strings.Split(card2, "-")[1]]
	})
	candidates := []string{}
	for _, card := range avCardsArr {
		parts := strings.Split(card, "-")
		t := parts[0]
		if (t == opdnType) && (opdnType != hokm) {
			continue
		}
		i, _ := strconv.ParseInt(parts[1], 10, 64)
		upperFound := false
		for j := i + 1; j <= 14; j++ {
			if !usedCards[fmt.Sprintf("%s-%d", t, j)] {
				upperFound = true
				break
			}
		}
		if !upperFound {
			candidates = append(candidates, card)
		}
	}
	// log.Println("----------------------")
	// log.Println(avCardsArr)
	// log.Println(candidates)
	// log.Println("----------------------")
	for _, card := range candidates {
		if strings.Split(card, "-")[0] != hokm {
			return card
		}
	}
	if tmdhType != "" {
		for _, card := range candidates {
			if strings.Split(card, "-")[0] == tmdhType {
				return card
			}
		}
	}
	if (len(candidates) > 0) && (float64(len(candidates)) >= math.Floor((float64(len(avCards)) / 2))) {
		return candidates[len(candidates)-1]
	}
	return ""
}

func getMostMinUpperValidSameTypeCard(thisRound []string, usedCards map[string]bool, avCards []string, typ string, hokm string, tmdhType string) string {
	avCardsArr := avCards[:]
	sort.Slice(avCardsArr, func(i, j int) bool {
		card1 := avCardsArr[i]
		card2 := avCardsArr[j]
		return mapOfNameToIndex[strings.Split(card1, "-")[1]] < mapOfNameToIndex[strings.Split(card2, "-")[1]]
	})
	cands := []string{}
	for _, card := range avCardsArr {
		parts := strings.Split(card, "-")
		t := parts[0]
		if typ == t {
			i, _ := strconv.ParseInt(parts[1], 10, 64)
			upperFound := false
			for j := i + 1; j <= 14; j++ {
				if !usedCards[fmt.Sprintf("%s-%d", t, j)] {
					upperFound = true
					break
				}
			}
			if !upperFound {
				cands = append(cands, card)
			}
		}
	}
	candidates := []string{}
	for _, card := range cands {
		suitable := true
		for _, op := range thisRound {
			valCand := mapOfNameToIndex[strings.Split(card, "-")[1]]
			valOp := mapOfNameToIndex[strings.Split(op, "-")[1]]
			if valCand < valOp {
				suitable = false
				break
			}
		}
		if suitable {
			candidates = append(candidates, card)
		}
	}
	// log.Println("----------------------")
	// log.Println(avCardsArr)
	// log.Println(candidates)
	// log.Println("----------------------")
	for _, card := range candidates {
		if strings.Split(card, "-")[0] != hokm {
			return card
		}
	}
	if tmdhType == typ {
		for _, card := range candidates {
			if strings.Split(card, "-")[0] == tmdhType {
				return card
			}
		}
	}
	if len(candidates) > 0 {
		return candidates[len(candidates)-1]
	}
	return ""
}

func getHokmCards(cards []string, hokm string) []string {
	hokmCards := []string{}
	for _, card := range cards {
		if strings.Split(card, "-")[0] == hokm {
			hokmCards = append(hokmCards, card)
		}
	}
	return hokmCards
}

func sortCards(cards []string, order string) []string {
	sort.Slice(cards, func(i, j int) bool {
		card1 := cards[i]
		card2 := cards[j]
		if order == "asc" {
			return mapOfNameToIndex[strings.Split(card1, "-")[1]] < mapOfNameToIndex[strings.Split(card2, "-")[1]]
		} else {
			return mapOfNameToIndex[strings.Split(card1, "-")[1]] > mapOfNameToIndex[strings.Split(card2, "-")[1]]
		}
	})
	return cards
}

func getMinIrrevalentTypeCard(cards []string, hokm string) string {
	minCard := ""
	for _, card := range cards {
		if strings.Split(card, "-")[0] != hokm {
			if minCard == "" {
				minCard = card
			} else {
				if mapOfNameToIndex[strings.Split(card, "-")[1]] < mapOfNameToIndex[strings.Split(minCard, "-")[1]] {
					minCard = card
				}
			}
		}
	}
	if minCard != "" {
		return minCard
	}
	for _, card := range cards {
		if minCard == "" {
			minCard = card
		} else {
			if mapOfNameToIndex[strings.Split(card, "-")[1]] > mapOfNameToIndex[strings.Split(minCard, "-")[1]] {
				minCard = card
			}
		}
	}
	return minCard
}

func getMinSameTypeCard(cards []string, t string, hokm string) string {
	if t == "" {
		testables := cards[:]
		sort.Slice(testables, func(i, j int) bool {
			card1 := testables[i]
			card2 := testables[j]
			t1 := strings.Split(card1, "-")[0]
			t2 := strings.Split(card2, "-")[0]
			if t1 == hokm {
				if t2 == hokm {
					return mapOfNameToIndex[strings.Split(card1, "-")[1]] < mapOfNameToIndex[strings.Split(card2, "-")[1]]
				} else {
					return false
				}
			} else {
				if t2 == hokm {
					return true
				} else {
					return mapOfNameToIndex[strings.Split(card1, "-")[1]] < mapOfNameToIndex[strings.Split(card2, "-")[1]]
				}
			}
		})
		nonHokms := []string{}
		for _, card := range testables {
			if strings.Split(card, "-")[0] != hokm {
				nonHokms = append(nonHokms, card)
			}
		}
		if len(nonHokms) > 0 {
			finals := []string{}
			for _, card := range nonHokms {
				if mapOfNameToIndex[strings.Split(card, "-")[1]] <= 3 {
					finals = append(finals, card)
				}
			}
			if len(finals) > 0 {
				return finals[rand.Intn(len(finals))]
			} else {
				return nonHokms[0]
			}
		} else {
			return testables[len(testables)-1]
		}
	} else {
		minCard := ""
		for _, card := range cards {
			if strings.Split(card, "-")[0] == t {
				if minCard == "" {
					minCard = card
				} else {
					if mapOfNameToIndex[strings.Split(card, "-")[1]] < mapOfNameToIndex[strings.Split(minCard, "-")[1]] {
						minCard = card
					}
				}
			}
		}
		return minCard
	}
}

func isBetterThan(card1 string, card2 string, stageType string, hokm string) bool {
	valCard1 := mapOfNameToIndex[strings.Split(card1, "-")[1]]
	valCard2 := mapOfNameToIndex[strings.Split(card2, "-")[1]]
	typeCard1 := strings.Split(card1, "-")[0]
	typeCard2 := strings.Split(card2, "-")[0]

	if stageType == hokm {
		if typeCard1 == hokm {
			if typeCard2 == hokm {
				if valCard1 > valCard2 {
					return true
				} else {
					return false
				}
			} else {
				return false
			}
		} else {
			if typeCard2 == hokm {
				return false
			} else {
				if valCard1 > valCard2 {
					return true
				} else {
					return false
				}
			}
		}
	}
	if typeCard1 == hokm {
		if typeCard2 == hokm {
			if valCard1 > valCard2 {
				return true
			} else {
				return false
			}
		} else {
			return true
		}
	} else if typeCard1 == stageType {
		if typeCard2 == hokm {
			return false
		} else if typeCard2 == stageType {
			if valCard1 > valCard2 {
				return true
			} else {
				return false
			}
		} else {
			return true
		}
	} else {
		if typeCard2 == hokm {
			return false
		} else if typeCard2 == stageType {
			return false
		} else {
			if valCard1 > valCard2 {
				return true
			} else {
				return false
			}
		}
	}
}

func betterThanTwoOthers(tmAction string, op1 string, op2 string, stageType string, hokm string) bool {
	if isBetterThan(tmAction, op1, stageType, hokm) && isBetterThan(tmAction, op2, stageType, hokm) {
		return true
	}
	return false
}

func chooseSuitableHokmCard(hokmCards []string, thisRound []string, avCards []string, stageType string, hokm string) string {
	sorted := sortCards(hokmCards[:], "asc")
	for _, hokmCard := range sorted {
		res := false
		for _, card := range thisRound {
			if isBetterThan(card, hokmCard, stageType, hokm) {
				res = true
				break
			}
		}
		if !res {
			return hokmCard
		}
	}
	return getMinIrrevalentTypeCard(avCards, hokm)
}

func isFirstPlayerWinning(card string, op1 string, usedCards map[string]bool, typ string, hokm string) bool {
	if isBetterThan(card, op1, typ, hokm) {
		parts := strings.Split(card, "-")
		t := parts[0]
		i, _ := strconv.ParseInt(parts[1], 10, 64)
		upperFound := false
		for j := i + 1; j <= 14; j++ {
			if !usedCards[fmt.Sprintf("%s-%d", t, j)] {
				upperFound = true
				break
			}
		}
		return !upperFound
	}
	return false
}

func logic(agent *GameHolder, opts []string) string {

	usedCards := map[string]bool{}

	avCards := []string{}
	if len(opts) > 0 {
		avCards = opts
	} else {
		for k, v := range agent.AvailableCards {
			if v {
				avCards = append(avCards, k)
			}
		}
	}

	// log.Println(avCards)

	for i, deck := range agent.Decks {
		if i == len(agent.Decks)-1 {
			continue
		}
		for _, card := range deck.Cards {
			usedCards[card] = true
		}
	}

	myIndex := len(agent.Decks[len(agent.Decks)-1].Cards)

	myTeamMate := 0

	if myIndex == 0 {
		myTeamMate = 2
	} else if myIndex == 1 {
		myTeamMate = 3
	} else if myIndex == 2 {
		myTeamMate = 0
	} else if myIndex == 3 {
		myTeamMate = 1
	}

	if myIndex > 0 {
		if myIndex == 3 {
			myTeamMateAction := agent.Decks[len(agent.Decks)-1].Cards[myTeamMate]
			op1 := agent.Decks[len(agent.Decks)-1].Cards[0]
			op2 := agent.Decks[len(agent.Decks)-1].Cards[2]
			if betterThanTwoOthers(myTeamMateAction, op1, op2, agent.StageType, agent.Hokm) {
				// log.Println("----------------------")
				// log.Println("team mate is winner")
				// log.Println("----------------------")
				minCard := getMinSameTypeCard(avCards, agent.StageType, agent.Hokm)
				if minCard != "" {
					return minCard
				} else {
					return getMinIrrevalentTypeCard(avCards, agent.Hokm)
				}
			} else {
				mostUpperValidCard := getMostMinUpperValidSameTypeCard(agent.Decks[len(agent.Decks)-1].Cards, usedCards, avCards, agent.StageType, agent.Hokm, agent.TmDnHave)
				if mostUpperValidCard != "" {
					return mostUpperValidCard
				} else {
					minCard := getMinSameTypeCard(avCards, agent.StageType, agent.Hokm)
					if minCard != "" {
						return minCard
					} else {
						hokmCards := getHokmCards(avCards, agent.Hokm)
						if len(hokmCards) > 0 {
							sorted := sortCards(hokmCards[:], "asc")
							for _, hokmCard := range sorted {
								if betterThanTwoOthers(hokmCard, agent.Decks[len(agent.Decks)-1].Cards[0], agent.Decks[len(agent.Decks)-1].Cards[2], agent.StageType, agent.Hokm) {
									return hokmCard
								}
							}
							return getMinIrrevalentTypeCard(avCards, agent.Hokm)
						} else {
							return getMinIrrevalentTypeCard(avCards, agent.Hokm)
						}
					}
				}
			}
		} else if myIndex == 2 {
			myTeamMateAction := agent.Decks[len(agent.Decks)-1].Cards[myTeamMate]
			if isFirstPlayerWinning(myTeamMateAction, agent.Decks[len(agent.Decks)-1].Cards[1], usedCards, agent.StageType, agent.Hokm) {
				// log.Println("----------------------")
				// log.Println("team mate is 'absolute' winner")
				// log.Println("----------------------")
				minCard := getMinSameTypeCard(avCards, agent.StageType, agent.Hokm)
				if minCard != "" {
					return minCard
				} else {
					return getMinIrrevalentTypeCard(avCards, agent.Hokm)
				}
			} else {
				mostUpperValidCard := getMostMinUpperValidSameTypeCard(agent.Decks[len(agent.Decks)-1].Cards, usedCards, avCards, agent.StageType, agent.Hokm, agent.TmDnHave)
				if mostUpperValidCard != "" {
					return mostUpperValidCard
				} else {
					minCard := getMinSameTypeCard(avCards, agent.StageType, agent.Hokm)
					if minCard != "" {
						return minCard
					} else {
						hokmCards := getHokmCards(avCards, agent.Hokm)
						if len(hokmCards) > 0 {
							return chooseSuitableHokmCard(hokmCards, agent.Decks[len(agent.Decks)-1].Cards, avCards, agent.StageType, agent.Hokm)
						} else {
							return getMinIrrevalentTypeCard(avCards, agent.Hokm)
						}
					}
				}
			}
		} else {
			mostUpperValidCard := getMostMinUpperValidSameTypeCard(agent.Decks[len(agent.Decks)-1].Cards, usedCards, avCards, agent.StageType, agent.Hokm, agent.TmDnHave)
			if mostUpperValidCard != "" {
				return mostUpperValidCard
			} else {
				minCard := getMinSameTypeCard(avCards, agent.StageType, agent.Hokm)
				if minCard != "" {
					return minCard
				} else {
					hokmCards := getHokmCards(avCards, agent.Hokm)
					if len(hokmCards) > 0 {
						return chooseSuitableHokmCard(hokmCards, agent.Decks[len(agent.Decks)-1].Cards, avCards, agent.StageType, agent.Hokm)
					} else {
						return getMinIrrevalentTypeCard(avCards, agent.Hokm)
					}
				}
			}
		}
	} else {
		mostUpperValidCard := getMostUpperValidCard(usedCards, avCards, agent.Hokm, agent.TmDnHave, agent.OpDnHave)
		if mostUpperValidCard != "" {
			return mostUpperValidCard
		} else {
			minCard := getMinSameTypeCard(avCards, "", agent.Hokm)
			if minCard != "" {
				return minCard
			} else {
				hokmCards := getHokmCards(avCards, agent.Hokm)
				if len(hokmCards) > 0 {
					sorted := sortCards(hokmCards[:], "dsc")
					if agent.Round > 3 {
						return sorted[0]
					} else {
						return sorted[len(sorted)-1]
					}
				} else {
					return getMinIrrevalentTypeCard(avCards, agent.Hokm)
				}
			}
		}
	}
}

func (h *HokmAgent) react(agent *GameHolder, choice string, choiceType string) {
	data := map[string]any{}
	if choiceType == "emoji" {
		data["emoji"] = choice
	} else {
		data["text"] = choice
	}
	innerData := map[string]any{
		"topicId":  agent.TopicId,
		"memberId": agent.MyMemberId,
		"data":     data,
	}
	future.Async(func() {
		time.Sleep(time.Duration(1) * time.Second)
		doHttpCall("/messages/create", "2", innerData)
	}, false)
}

func (h *HokmAgent) play(agent *GameHolder) {

	agent.Simulator = []string{}

	future.Async(func() {

		action := logic(agent, []string{})
		// log.Println(action)

		time.Sleep(time.Duration(2) * time.Second)

		h.sendPacket(agent, action)
	}, false)
}

var Queues *cmap.ConcurrentMap[string, chan func()]

func (h *HokmAgent) OnTopicSend(input models.Send) any {

	var data = map[string]any{}
	err := json.Unmarshal([]byte(input.Data), &data)
	if err != nil {
		return map[string]any{"error": err.Error()}
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

	q, okq := Queues.Get(input.Topic.Id)
	if !okq {
		return map[string]any{}
	}

	q <- func() {
		agent := new(GameHolder)
		if input.Action == "single" {
			gameRecord := AgentRecord{Id: input.TargetMember.Id}
			e := h.Store.Db().First(&gameRecord).Error
			if e != nil {
				log.Println(e)
				return
			}
			errM := json.Unmarshal([]byte(gameRecord.Data), agent)
			if errM != nil {
				log.Println(errM)
				return
			}
		}

		switch key {
		case "suggestReact":
			{
				choice := data["choice"].(string)
				choiceType := data["choiceType"].(string)
				h.react(agent, choice, choiceType)
				break
			}
		case "destruct":
			{
				h.Store.Db().Delete(&AgentRecord{}, "topic_id = ?", input.Topic.Id)
				if Queues.Has(input.Topic.Id) {
					Queues.Remove(input.Topic.Id)
				}
				break
			}
		case "gameCreation":
			{
				inp := extractData[exportedGame](data["game"])
				agent.ExpGame = &inp
				agent.AvailableCards = map[string]bool{}
				agent.Decks = []*Deck{{Cards: []string{}}}
				agent.TeamId = data["teamId"].(string)
				for _, option := range agent.ExpGame.Cards {
					agent.AvailableCards[option] = true
				}
				agent.EmojiCount = 0
				agent.TextCount = 0
				agent.TmDnHave = ""
				agent.OpDnHave = ""
				h.SaveAgent(agent)
				break
			}
		case "simGame":
			{
				options := data["options"].([]interface{})
				opts := []string{}
				for _, o := range options {
					opts = append(opts, o.(string))
				}
				action := logic(agent, opts)
				h.SaveAgent(agent)
				h.SendTopicPacket("single", agent.SpaceId, agent.TopicId, agent.MyMemberId, agent.GodMemberId,
					map[string]any{
						"type":          "resSimGame",
						"action":        action,
						"humanPlayerId": data["humanPlayerId"].(string),
					})
				break
			}
		case "startGame":
			{
				h.play(agent)
				break
			}
		case "tellMeHokm":
			{
				future.Async(func() {

					time.Sleep(time.Duration(5) * time.Second)

					firstCards := agent.ExpGame.Cards[0:5]
					cardCount := []int{0, 0, 0, 0}
					for _, card := range firstCards {
						for i, ct := range cardTypes {
							if ct == strings.Split(card, "-")[0] {
								cardCount[i]++
								break
							}
						}
					}

					maxIndex := 0
					for i := range cardTypes {
						if cardCount[maxIndex] <= cardCount[i] {
							maxIndex = i
						}
					}
					h.SendTopicPacket("single", agent.SpaceId, agent.TopicId, agent.MyMemberId, agent.GodMemberId,
						map[string]any{
							"type": "specifyHokm",
							"hokm": cardTypes[maxIndex],
						})
				}, false)
				break
			}
		case "hokmSpecification":
			{
				agent.Hokm = data["hokm"].(string)
				h.SaveAgent(agent)
				break
			}
		case "gamePlay":
			{
				a := data["action"].(string)
				pmi := data["playerMemberId"].(string)
				if agent.MyMemberId == pmi {
					agent.AvailableCards[a] = false
				}
				t := strings.Split(a, "-")[0]
				if (t != agent.Hokm) && (t != agent.StageType) {
					for i, p := range agent.ExpGame.Players {
						if p.MemberId == pmi {
							tm := 0
							if i >= 2 {
								tm = i - 2
							} else {
								tm = i + 2
							}
							if agent.ExpGame.Players[tm].MemberId == agent.MyMemberId {
								agent.TmDnHave = t
							}
							ops := []int{0, 0}
							if (i == 0) || (i == 2) {
								ops = []int{1, 3}
							} else if (i == 1) || (i == 3) {
								ops = []int{0, 2}
							}
							for _, op := range ops {
								if agent.ExpGame.Players[op].MemberId == agent.MyMemberId {
									agent.OpDnHave = t
								}
							}
							h.SaveAgent(agent)
							break
						}
					}
				}
				if agent.StageType == "" {
					agent.StageType = strings.Split(a, "-")[0]
				}
				if !agent.CurrentDeck[a] {
					agent.CurrentDeck[a] = true
					agent.Decks[len(agent.Decks)-1].Cards = append(agent.Decks[len(agent.Decks)-1].Cards, a)
					agent.Round = data["round"].(float64)
				}
				h.SaveAgent(agent)
				break
			}
		case "stageResult":
			{
				if agent.StageType != "" {
					agent.CurrentDeck = map[string]bool{}
					agent.StageType = ""
					agent.Decks = append(agent.Decks, &Deck{Cards: []string{}})
					h.SaveAgent(agent)
				}
				break
			}
		}
	}

	return map[string]any{}
}

var token = ""

// var address = "185.204.168.179:8080"
// var protocol = ""

var address = "game.midopia.com"
var protocol = "s"

// var address = "localhost:8080"

func doHttpCall(path string, layer string, val any) {
	future.Async(func() {
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
	}, false)
}

func (h *HokmAgent) SaveAgent(g *GameHolder) {
	gameStr, err := json.Marshal(g)
	if err != nil {
		log.Println(err)
		return
	}
	err2 := h.Store.Db().Save(&AgentRecord{Id: g.MyMemberId, TopicId: g.TopicId, Data: string(gameStr)}).Error
	if err2 != nil {
		log.Println(err2)
	}
}

func (h *HokmAgent) SendTopicPacket(typ string, spaceId string, topicId string, memberId string, recvId string, data any) {
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
