package module_core

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"kasper/cmd/babble/sigma/abstract"
	moduleactor "kasper/cmd/babble/sigma/core/module/actor"
	modulecoremodel "kasper/cmd/babble/sigma/core/module/core/model"
	"kasper/cmd/babble/sigma/core/module/core/model/worker"
	"kasper/cmd/babble/sigma/layer1/adapters"
	moduleactormodel "kasper/cmd/babble/sigma/layer1/module/actor"
	mach_model "kasper/cmd/babble/sigma/machiner/model"
	"kasper/cmd/babble/sigma/utils/crypto"
	"kasper/cmd/babble/sigma/utils/future"
	"kasper/src/babble"
	"log"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"kasper/cmd/babble/sigma/layer2/model"

	"kasper/src/proxy/inmem"
)

type ResponseHolder struct {
	Payload []byte
	Effects abstract.Effects
}

type Election struct {
	MyNum        string
	Participants map[string]bool
	Commits      map[string][]byte
	Reveals      map[string]string
}

type Core struct {
	lock           sync.Mutex
	id             string
	utils          abstract.IUtils
	gods           []string
	layers         []abstract.ILayer
	actor          *moduleactor.Actor
	chain          chan any
	chainCallbacks map[string]*ChainCallback
	babbleInst     *babble.Babble
	Ip             string
	Elections      []Election
	ElecReg        bool
	ElecStarter    string
	ElecStartTime  int64
	Executors      map[string]bool
	appPendingTrxs []*worker.Trx
}

var MAX_VALIDATOR_COUNT = 5

type ChainCallback struct {
	Fn        func([]byte, int, error)
	Executors map[string]bool
	Responses map[string]string
}

func NewCore(id string, log abstract.Log) abstract.ICore {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	execs := map[string]bool{}
	execs[localAddr] = true
	return &Core{
		id:             id,
		utils:          modulecoremodel.NewUtils(log),
		gods:           make([]string, 0),
		layers:         make([]abstract.ILayer, 0),
		actor:          moduleactor.NewActor(),
		chain:          nil,
		chainCallbacks: map[string]*ChainCallback{},
		babbleInst:     nil,
		Ip:             localAddr,
		Elections:      nil,
		ElecReg:        false,
		Executors:      execs,
	}
}

func (c *Core) Push(l abstract.ILayer) {
	c.layers = append(c.layers, l)
}

func (c *Core) Get(index int) abstract.ILayer {
	return c.layers[index-1]
}

func (c *Core) Id() string {
	return c.id
}

func (c *Core) Layers() []abstract.ILayer {
	return c.layers
}

func (c *Core) Chain() *babble.Babble {
	return c.babbleInst
}

func (c *Core) Gods() []string {
	return c.gods
}

func (c *Core) AppPendingTrxs() {
	elpisTrxs := []*worker.Trx{}
	wasmTrxs := []*worker.Trx{}
	for _, trx := range c.appPendingTrxs {
		if trx.Runtime == "elpis" {
			elpisTrxs = append(elpisTrxs, trx)
		} else if trx.Runtime == "wasm" {
			wasmTrxs = append(wasmTrxs, trx)
		}
	}
	if len(elpisTrxs) > 0 {
		abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools()).Elpis().ExecuteChainTrxsGroup(elpisTrxs)
	}
	if len(wasmTrxs) > 0 {
		abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools()).Wasm().ExecuteChainTrxsGroup(wasmTrxs)
	}
	c.appPendingTrxs = []*worker.Trx{}
}

func (c *Core) ClearAppPendingTrxs() {
	c.appPendingTrxs = []*worker.Trx{}
}

func (c *Core) ExecAppletRequestOnChain(machineId string, key string, packet []byte, userId string, callback func([]byte, int, error)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	callbackId := crypto.SecureUniqueString()
	c.chainCallbacks[callbackId] = &ChainCallback{Fn: callback, Executors: map[string]bool{}, Responses: map[string]string{}}
	vm := mach_model.Vm{MachineId: machineId}
	abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools()).Storage().Db().First(&vm)
	c.chain <- abstract.ChainPacket{Type: "request", Meta: map[string]any{"requester": c.Ip, "origin": c.id, "requestId": callbackId, "isBase": false, "runtime": vm.Runtime, "userId": userId, "machineId": machineId}, Key: key, Payload: packet, Effects: abstract.Effects{DbUpdates: []abstract.Update{}, CacheUpdates: []abstract.CacheUpdate{}}}
}

func (c *Core) ExecBaseRequestOnChain(key string, packet any, layer int, token string, callback func([]byte, int, error)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	callbackId := crypto.SecureUniqueString()
	c.chainCallbacks[callbackId] = &ChainCallback{Fn: callback, Executors: map[string]bool{}, Responses: map[string]string{}}
	serialized, err := json.Marshal(packet)
	if err == nil {
		c.chain <- abstract.ChainPacket{Type: "request", Meta: map[string]any{"requester": c.Ip, "origin": c.id, "requestId": callbackId, "isBase": true, "layer": layer, "token": token}, Key: key, Payload: serialized, Effects: abstract.Effects{DbUpdates: []abstract.Update{}, CacheUpdates: []abstract.CacheUpdate{}}}
	} else {
		log.Println(err)
	}
}

func (c *Core) ExecBaseResponseOnChain(callbackId string, packet any, resCode int, e string, updates []abstract.Update, cacheUpdates []abstract.CacheUpdate) {
	serialized, err := json.Marshal(packet)
	if err == nil {
		c.chain <- abstract.ChainPacket{Type: "response", Meta: map[string]any{"executor": c.Ip, "requestId": callbackId, "isBase": true, "responseCode": resCode, "error": e}, Payload: serialized, Effects: abstract.Effects{DbUpdates: updates, CacheUpdates: cacheUpdates}}
	} else {
		log.Println(err)
	}
}

func (c *Core) ExecAppletResponseOnChain(callbackId string, packet []byte, resCode int, e string, updates []abstract.Update, cacheUpdates []abstract.CacheUpdate) {
	c.chain <- abstract.ChainPacket{Type: "response", Meta: map[string]any{"executor": c.Ip, "requestId": callbackId, "isBase": false, "responseCode": resCode, "error": e}, Payload: packet, Effects: abstract.Effects{DbUpdates: updates, CacheUpdates: cacheUpdates}}
}

func (c *Core) OnChainPacket(packet abstract.ChainPacket) {
	c.lock.Lock()
	defer c.lock.Unlock()
	switch packet.Type {
	case "election":
		{
			if packet.Key == "choose-validator" {
				phaseRaw, ok := packet.Meta["phase"]
				if !ok {
					return
				}
				phase, ok2 := phaseRaw.(string)
				if !ok2 {
					return
				}
				voterRaw, ok := packet.Meta["voter"]
				if !ok {
					return
				}
				voter, ok2 := voterRaw.(string)
				if !ok2 {
					return
				}
				if c.Elections == nil {
					c.Elections = []Election{}
				}
				if phase == "start-reg" {
					c.ElecReg = true
					c.ElecStarter = voter
					c.ElecStartTime = time.Now().UnixMilli()
					for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
						c.Elections = append(c.Elections, Election{Participants: map[string]bool{}, Commits: map[string][]byte{}, Reveals: map[string]string{}})
					}
					c.chain <- abstract.ChainPacket{
						Type:    "election",
						Key:     "choose-validator",
						Meta:    map[string]any{"phase": "register", "voter": c.Ip},
						Payload: []byte("{}"),
					}
					if voter == c.Ip {
						future.Async(func() {
							time.Sleep(time.Duration(10) * time.Second)
							c.chain <- abstract.ChainPacket{
								Type:    "election",
								Key:     "choose-validator",
								Meta:    map[string]any{"phase": "end-reg", "voter": c.Ip},
								Payload: []byte("{}"),
							}
						}, false)
					}
				} else if phase == "end-reg" {
					if c.ElecStarter == voter && ((time.Now().UnixMilli() - c.ElecStartTime) > 8000) {
						c.ElecReg = false
						payload := [][]byte{}
						nodeCount := c.babbleInst.Peers.Len()
						hasher := sha256.New()
						for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
							r := fmt.Sprintf("%d", rand.Intn(nodeCount))
							c.Elections[i].MyNum = r
							hasher.Write([]byte(r))
							bs := hasher.Sum(nil)
							payload = append(payload, bs)
						}
						data, _ := json.Marshal(payload)
						c.chain <- abstract.ChainPacket{
							Type:    "election",
							Key:     "choose-validator",
							Meta:    map[string]any{"phase": "commit", "voter": c.Ip},
							Payload: data,
						}
					}
				} else if phase == "register" {
					if c.ElecReg {
						for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
							c.Elections[i].Participants[voter] = true
						}
					}
				} else if phase == "commit" {
					votes := [][]byte{}
					e := json.Unmarshal(packet.Payload, &votes)
					if e != nil {
						return
					}
					if len(votes) < MAX_VALIDATOR_COUNT {
						return
					}
					if c.Elections[0].Participants[voter] {
						for i := 0; i < min(MAX_VALIDATOR_COUNT, c.babbleInst.Peers.Len()); i++ {
							c.Elections[i].Commits[voter] = votes[i]
						}
						if len(c.Elections[0].Commits) == len(c.Elections[0].Participants) {
							myReveals := []string{}
							for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
								myReveals = append(myReveals, c.Elections[i].MyNum)
							}
							data, _ := json.Marshal(myReveals)
							c.chain <- abstract.ChainPacket{
								Type:    "election",
								Key:     "choose-validator",
								Meta:    map[string]any{"phase": "reveal", "voter": c.Ip},
								Payload: data,
							}
						}
					}
				} else if phase == "reveal" {
					votes := []string{}
					e := json.Unmarshal(packet.Payload, &votes)
					if e != nil {
						return
					}
					if len(votes) < MAX_VALIDATOR_COUNT {
						return
					}
					if c.Elections[0].Participants[voter] {
						for i := 0; i < min(MAX_VALIDATOR_COUNT, c.babbleInst.Peers.Len()); i++ {
							c.Elections[i].Reveals[voter] = votes[i]
						}
						if len(c.Elections[0].Reveals) == len(c.Elections[0].Participants) {
							c.Executors = map[string]bool{}
							nodesArr := []string{}
							for p := range c.Elections[0].Participants {
								nodesArr = append(nodesArr, p)
							}
							sort.Strings(nodesArr)
							for _, elec := range c.Elections[0:min(MAX_VALIDATOR_COUNT, len(nodesArr))] {
								res := -1
								first := true
								for v := range elec.Participants {
									hasher := sha256.New()
									commit := elec.Commits[v]
									reveal := elec.Reveals[v]
									hasher.Write([]byte(reveal))
									bs := hasher.Sum(nil)
									if !bytes.Equal(bs, commit) {
										continue
									}
									num, e := strconv.ParseInt(reveal, 10, 32)
									if e != nil {
										continue
									}
									if first {
										first = false
										res = int(num)
									} else {
										res ^= int(num)
									}
								}
								result := res % len(nodesArr)
								candidate := nodesArr[result]
								c.Executors[candidate] = true
								nodesArr = append(nodesArr[:result], nodesArr[result+1:]...)
								c.Elections = nil
							}
						}
					}
				}
			}
			break
		}
	case "request":
		{
			requesterRaw, ok := packet.Meta["requester"]
			if !ok {
				return
			}
			requester, ok2 := requesterRaw.(string)
			if !ok2 {
				return
			}
			requestIdRaw, ok := packet.Meta["requestId"]
			if !ok {
				return
			}
			requestId, ok2 := requestIdRaw.(string)
			if !ok2 {
				return
			}
			execs := map[string]bool{}
			for k, v := range c.Executors {
				execs[k] = v
			}
			if requester == c.Ip {
				c.chainCallbacks[requestId].Executors = execs
			} else {
				c.chainCallbacks[requestId] = &ChainCallback{Fn: nil, Executors: execs, Responses: map[string]string{}}
			}
			if !c.Executors[c.Ip] {
				return
			}
			isBaseRaw, ok := packet.Meta["isBase"]
			if !ok {
				return
			}
			isBase, ok2 := isBaseRaw.(bool)
			if !ok2 {
				return
			}
			if isBase {
				layerRaw, ok := packet.Meta["layer"]
				if !ok {
					return
				}
				layer, ok2 := layerRaw.(float64)
				if !ok2 {
					return
				}
				originRaw, ok := packet.Meta["origin"]
				if !ok {
					return
				}
				origin, ok2 := originRaw.(string)
				if !ok2 {
					return
				}
				tokenRaw, ok := packet.Meta["token"]
				if !ok {
					return
				}
				token, ok2 := tokenRaw.(string)
				if !ok2 {
					return
				}
				l := c.Get(int(layer))
				if l == nil {
					return
				}
				action := l.Actor().FetchAction(packet.Key)
				if action == nil {
					return
				}
				var input abstract.IInput
				i, err2 := action.(*moduleactormodel.SecureAction).ParseInput("fed", string(packet.Payload))
				if err2 != nil {
					log.Println(err2)
					c.ExecBaseResponseOnChain(requestId, abstract.EmptyPayload{}, 400, "input parsing error", []abstract.Update{}, []abstract.CacheUpdate{})
					return
				}
				input = i
				action.(abstract.ISecureAction).SecurlyActChain(l, token, requestId, input, origin)
			} else {
				userIdRaw, ok := packet.Meta["userId"]
				if !ok {
					return
				}
				userId, ok2 := userIdRaw.(string)
				if !ok2 {
					return
				}
				machineIdRaw, ok := packet.Meta["machineId"]
				if !ok {
					return
				}
				machineId, ok2 := machineIdRaw.(string)
				if !ok2 {
					return
				}
				runtimeRaw, ok := packet.Meta["runtime"]
				if !ok {
					return
				}
				runtimeId, ok2 := runtimeRaw.(string)
				if !ok2 {
					return
				}
				c.appPendingTrxs = append(c.appPendingTrxs, &worker.Trx{CallbackId: requestId, Runtime: runtimeId, UserId: userId, MachineId: machineId, Key: packet.Key, Payload: string(packet.Payload)})
			}
			break
		}
	case "response":
		{
			execitorAddrRaw, ok := packet.Meta["executor"]
			if !ok {
				return
			}
			execitorAddr, ok2 := execitorAddrRaw.(string)
			if !ok2 {
				return
			}
			callbackIdRaw, ok := packet.Meta["requestId"]
			if !ok {
				return
			}
			callbackId, ok2 := callbackIdRaw.(string)
			if !ok2 {
				return
			}
			callback, ok3 := c.chainCallbacks[callbackId]
			if ok3 {
				if !callback.Executors[execitorAddr] {
					return
				}
				str, _ := json.Marshal(ResponseHolder{Payload: packet.Payload, Effects: packet.Effects})
				callback.Responses[execitorAddr] = string(str)
				if len(callback.Responses) < len(callback.Executors) {
					return
				}
				temp := ""
				for _, res := range callback.Responses {
					if temp == "" {
						temp = res
					} else if res != temp {
						temp = ""
						break
					}
				}
				if temp == "" {
					return
				}
				if !callback.Executors[c.Ip] {
					tb := abstract.UseToolbox[*module_model.ToolboxL2](c.Get(2).Tools())
					kvstoreKeyword := "kvstore: "
					tb.Storage().DoTrx(func(trx adapters.ITrx) error {
						for _, ef := range packet.Effects.DbUpdates {
							if (len(ef.Data) > len(kvstoreKeyword)) && (ef.Data[0:len(kvstoreKeyword)] == kvstoreKeyword) {
								tb.Wasm().ExecuteChainEffects(ef.Data[0:len(kvstoreKeyword)])
							} else {
								trx.Db().Exec(ef.Data)
							}
						}
						return nil
					})
					for _, ef := range packet.Effects.CacheUpdates {
						if ef.Typ == "put" {
							tb.Cache().Put(ef.Key, ef.Val)
						} else if ef.Typ == "del" {
							tb.Cache().Del(ef.Key)
						}
					}
				}
				delete(c.chainCallbacks, callbackId)
				if callback.Fn != nil {
					resCodeRaw, ok := packet.Meta["responseCode"]
					if !ok {
						return
					}
					resCode, ok2 := resCodeRaw.(float64)
					if !ok2 {
						return
					}
					errCodeRaw, ok3 := packet.Meta["error"]
					if !ok3 {
						return
					}
					errCode, ok4 := errCodeRaw.(string)
					if !ok4 {
						return
					}
					if errCode == "" {
						callback.Fn(packet.Payload, int(resCode), nil)
					} else {
						callback.Fn(packet.Payload, int(resCode), errors.New(errCode))
					}
				}
			}
			break
		}
	}
}

func (c *Core) NewHgHandler() *abstract.HgHandler {
	return &abstract.HgHandler{
		Sigma: c,
	}
}

func (c *Core) Load(gods []string, layers []abstract.ILayer, args []interface{}) {
	c.gods = gods
	c.layers = layers
	var output = args
	for i := len(layers) - 1; i >= 0; i-- {
		output = layers[i].BackFill(c, output...)
	}

	engine := output[0].(*babble.Babble)
	proxy := output[1].(*inmem.InmemProxy)

	c.chain = make(chan any, 1)
	future.Async(func() {
		for {
			op := <-c.chain
			serialized, err := json.Marshal(op)
			if err == nil {
				log.Println(string(serialized))
				proxy.SubmitTx(serialized)
			} else {
				log.Println(err)
			}
		}
	}, true)

	var sb abstract.IStateBuilder
	for i := 0; i < len(layers); i++ {
		sb = layers[i].InitSb(sb)
		if i > 0 {
			layers[i].ForFill(c, layers[i-1].Tools())
		} else {
			layers[i].ForFill(c)
		}
	}

	c.babbleInst = engine
}

func (c *Core) Utils() abstract.IUtils {
	return c.utils
}

func (c *Core) DoElection() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ElecReg = true
	c.chain <- abstract.ChainPacket{
		Type:    "election",
		Key:     "choose-validator",
		Meta:    map[string]any{"phase": "start-reg", "voter": c.Ip},
		Payload: []byte("{}"),
	}
}

func (c *Core) Run() {

	future.Async(func() {
		c.babbleInst.Run()
	}, false)

	future.Async(func() {
		electionStarted := false
		for {
			time.Sleep(time.Duration(1) * time.Second)
			minutes := time.Now().Minute()
			seconds := time.Now().Second()
			if (minutes == 0) && ((seconds >= 0) && (seconds <= 2)) {
				if !electionStarted {
					electionStarted = true
					c.DoElection()
					future.Async(func() {
						time.Sleep(time.Duration(5) * time.Second)
						electionStarted = false
					}, true)
				}
			}
		}
	}, true)
}
