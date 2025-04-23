package net_ws

import (
	"encoding/json"
	"errors"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/action"
	"kasper/src/abstract/models/input"
	module_model "kasper/src/shell/layer1/model"

	"log"

	"strings"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type WebsocketAnswer struct {
	Status    int
	RequestId string
	Data      any
}

type WsServer struct {}

func (q *Queue) AnswerSocket(conn *websocket.Conn, t string, requestId string, answer any) {
	q.Lock.Lock()
	defer q.Lock.Unlock()
	answerBytes, err0 := json.Marshal(answer)
	if err0 != nil {
		log.Println(err0)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(t+" "+requestId+" "+string(answerBytes))); err != nil {
		log.Println(err)
		return
	}
}

func ParseInput[T input.IInput](i string) (input.IInput, error) {
	body := new(T)
	err := json.Unmarshal([]byte(i), body)
	if err != nil {
		return nil, errors.New("invalid input format")
	}
	return *body, nil
}

func (ws *WsServer) PrepareAnswer(answer any) []byte {
	answerBytes, err0 := json.Marshal(answer)
	if err0 != nil {
		log.Println(err0)
		return nil
	}
	return answerBytes
}

type Queue struct {
	Lock sync.Mutex
	Ack  bool
	Data []any
}

var queue = map[string]*Queue{}

var cfg = websocket.Config{
	RecoverHandler: func(conn *websocket.Conn) {
		if err := recover(); err != nil {
			log.Println(err)
		}
	},
}

func connectListener(ws *WsServer, sig signaler.ISignaler, uid string, conn *websocket.Conn) (*signaler.Listener, *Queue) {
	sig.Lock()
	defer sig.Unlock()
	var lis *signaler.Listener
	lisTemp, found := sig.Listeners().Get(uid)
	if found {
		lis = lisTemp
	}
	q := queue[uid]
	if lis == nil {
		q = &Queue{Data: []any{}, Ack: true}
		queue[uid] = q
	}
	lis = &signaler.Listener{
		Id:      uid,
		Paused:  false,
		DisTime: 0,
		Signal: func(b any) {
			q.Lock.Lock()
			defer q.Lock.Unlock()
			if b != nil {
				q.Data = append(q.Data, b)
			}
			if len(q.Data) > 0 {
				if q.Ack {
					q.Ack = false
					err := conn.WriteMessage(websocket.TextMessage, q.Data[0].([]byte))
					if err != nil {
						q.Ack = true
						log.Println(err)
					}
				}
			}
		},
	}
	q.Ack = true
	return lis, q
}

func (ws *WsServer) Load(actor action.IActor, httpServer network.IHttp, security security.ISecurity, sig signaler.ISignaler, storage storage.IStorage) {
	httpServer.Server().Get("/ws", websocket.New(func(conn *websocket.Conn) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
			}
		}()
		var uid string = ""
		var lis *signaler.Listener
		var q *Queue
		processPacket := func() int {
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			_, p, err := conn.ReadMessage()
			if err != nil {
				return 1
			}
			var dataStr = string(p[:])
			// log.Println(dataStr)
			if dataStr == "packet_received" {
				send := func() {
					q.Lock.Lock()
					defer q.Lock.Unlock()
					q.Ack = true
					if len(q.Data) > 0 {
						q.Data = q.Data[1:]
						if len(q.Data) > 0 {
							q.Ack = false
							err := conn.WriteMessage(websocket.TextMessage, q.Data[0].([]byte))
							if err != nil {
								q.Ack = true
								log.Println(err)
							}
						}
					}
				}
				send()
				return 0
			}
			var splittedMsg = strings.Split(dataStr, " ")
			var uri = splittedMsg[0]
			if len(splittedMsg) > 1 {
				if uri == "authenticate" {
					if len(splittedMsg) >= 3 {
						var userId = splittedMsg[1]
						var binData = []byte(splittedMsg[2])
						var signature = splittedMsg[3]
						var requestId = splittedMsg[4]
						success, _, _ := security.AuthWithSignature(userId, binData, signature)
						if success {
							uid = userId
							lis, q = connectListener(ws, sig, uid, conn)
							// var members []model.Member
							// storage.DoTrx(func(trx adapters.ITrx) error {
							// 	return trx.Db().Model(&model.Member{}).Where("user_id = ?", userId).Find(&members).Error
							// })
							// for _, member := range members {
							// 	sig.JoinGroup(member.SpaceId, uid)
							// }
							q.AnswerSocket(conn, "response", requestId, module_model.ResponseSimpleMessage{Message: "authenticated"})
							oldQueueEndPack, _ := json.Marshal(module_model.ResponseSimpleMessage{Message: "old_queue_end"})
							sig.ListenToSingle(lis)
							lis.Signal(oldQueueEndPack)
						} else {
							q.AnswerSocket(conn, "error", requestId, module_model.ResponseSimpleMessage{Message: "authentication failed"})
						}
					}
				} else {
					// if len(splittedMsg) >= 4 {
					// 	var requestId = splittedMsg[2]
					// 	var layerNumStr = splittedMsg[3]
					// 	layerNum, err := strconv.Atoi(layerNumStr)
					// 	if err != nil {
					// 		log.Println(err)
					// 		layerNum = 1
					// 		return 0
					// 	}
					// 	var body = dataStr[(len(uri) + 1 + len(requestId) + 1 + len(layerNumStr)):]
					// 	layer := core.Get(layerNum)
					// 	action := layer.Actor().FetchAction(uri)
					// 	if action == nil {
					// 		q.AnswerSocket(conn, "error", requestId, module_model.ResponseSimpleMessage{Message: "action not found"})
					// 		return 0
					// 	}
					// 	input, err := action.(*moduleactormodel.SecureAction).ParseInput("ws", body)
					// 	if err != nil {
					// 		log.Println(err)
					// 		q.AnswerSocket(conn, "error", requestId, module_model.ResponseSimpleMessage{Message: "parsing input failed"})
					// 		return 0
					// 	}
					// 	res, _, err2 := action.(*moduleactormodel.SecureAction).SecurelyAct(layer, ws.Tokens[uid], requestId, input, "")
					// 	if err2 != nil {
					// 		q.AnswerSocket(conn, "error", requestId, module_model.BuildErrorJson(err2.Error()))
					// 	} else {
					// 		q.AnswerSocket(conn, "response", requestId, ws.PrepareAnswer(res))
					// 	}
					// }
				}
			}
			return 0
		}
		for {
			if processPacket() != 0 {
				if lis != nil {
					lis.Paused = true
					lis.DisTime = time.Now().UnixMilli()
					time.Sleep(time.Duration(62) * time.Second)
					dispose := func() {
						sig.Lock()
						defer sig.Unlock()
						l, found := sig.Listeners().Get(uid)
						if !found {
							delete(queue, uid)
							sig.Listeners().Remove(uid)
							log.Println("client queue disposed")
						} else if l.Paused {
							if (time.Now().UnixMilli() - l.DisTime) > (60 * 1000) {
								delete(queue, uid)
								sig.Listeners().Remove(uid)
								log.Println("client queue disposed")
							}
						}
					}
					dispose()
					log.Println("socket broken")
				}
				break
			}
		}
	}, cfg))
}

func New(actor action.IActor, httpServer network.IHttp, security security.ISecurity, signaler signaler.ISignaler, storage storage.IStorage) *WsServer {
	ws := &WsServer{}
	ws.Load(actor, httpServer, security, signaler, storage)
	return ws
}
