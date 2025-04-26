package module_core

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"kasper/src/abstract/adapters/docker"
	"kasper/src/abstract/adapters/elpis"
	"kasper/src/abstract/adapters/file"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/security"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/adapters/tools"
	"kasper/src/abstract/adapters/wasm"
	iaction "kasper/src/abstract/models/action"
	"kasper/src/abstract/models/chain"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/info"
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/models/trx"
	"kasper/src/abstract/models/update"
	"kasper/src/abstract/models/worker"
	"kasper/src/abstract/state"
	"kasper/src/babble"
	actor "kasper/src/core/module/actor"
	mainstate "kasper/src/core/module/actor/model/state"
	module_trx "kasper/src/core/module/actor/model/trx"
	mach_model "kasper/src/shell/machiner/model"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"

	driver_docker "kasper/src/drivers/docker"
	driver_elpis "kasper/src/drivers/elpis"
	driver_file "kasper/src/drivers/file"
	driver_network "kasper/src/drivers/network"
	driver_security "kasper/src/drivers/security"
	driver_signaler "kasper/src/drivers/signaler"
	driver_storage "kasper/src/drivers/storage"
	driver_wasm "kasper/src/drivers/wasm"

	driver_network_fed "kasper/src/drivers/network/federation"

	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	mathrand "math/rand"

	cryp "crypto"
	"crypto/rand"

	"kasper/src/proxy/inmem"
)

type Tools struct {
	security security.ISecurity
	signaler signaler.ISignaler
	storage  storage.IStorage
	network  network.INetwork
	file     file.IFile
	wasm     wasm.IWasm
	elpis    elpis.IElpis
	docker   docker.IDocker
}

func (t *Tools) Security() security.ISecurity {
	return t.security
}

func (t *Tools) Signaler() signaler.ISignaler {
	return t.signaler
}

func (t *Tools) Storage() storage.IStorage {
	return t.storage
}

func (t *Tools) Network() network.INetwork {
	return t.network
}

func (t *Tools) File() file.IFile {
	return t.file
}

func (t *Tools) Wasm() wasm.IWasm {
	return t.wasm
}

func (t *Tools) Elpis() elpis.IElpis {
	return t.elpis
}

func (t *Tools) Docker() docker.IDocker {
	return t.docker
}

type Core struct {
	lock           sync.Mutex
	id             string
	tools          tools.ITools
	gods           []string
	chain          chan any
	chainCallbacks map[string]*chain.ChainCallback
	babbleInst     *babble.Babble
	Ip             string
	elections      []chain.Election
	elecReg        bool
	elecStarter    string
	elecStartTime  int64
	executors      map[string]bool
	appPendingTrxs []*worker.Trx
	actionStore    iaction.IActor
	privKey        *rsa.PrivateKey
}

var MAX_VALIDATOR_COUNT = 5

func NewCore(_ string) *Core {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	id := localAddr
	execs := map[string]bool{}
	execs["172.77.5.1"] = true
	execs["172.77.5.2"] = true
	return &Core{
		id:             id,
		gods:           make([]string, 0),
		chain:          nil,
		chainCallbacks: map[string]*chain.ChainCallback{},
		babbleInst:     nil,
		Ip:             localAddr,
		elections:      nil,
		elecReg:        false,
		executors:      execs,
		actionStore:    actor.NewActor(),
	}
}

func (c *Core) Actor() iaction.IActor {
	return c.actionStore
}

func (c *Core) ModifyStateSecurly(readonly bool, info info.IInfo, fn func(state.IState)) {
	trx := module_trx.NewTrx(c, c.Tools().Storage(), readonly)
	defer trx.Commit()
	s := mainstate.NewState(info, trx)
	fn(s)
}

func (c *Core) ModifyState(readonly bool, fn func(trx.ITrx)) {
	trx := module_trx.NewTrx(c, c.Tools().Storage(), readonly)
	defer trx.Commit()
	fn(trx)
}

func (c *Core) Tools() tools.ITools {
	return c.tools
}

func (c *Core) Id() string {
	return c.id
}

func (c *Core) Chain() *babble.Babble {
	return c.babbleInst
}

func (c *Core) Gods() []string {
	return c.gods
}

func (c *Core) IpAddr() string {
	return c.Ip
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
		c.Tools().Elpis().ExecuteChainTrxsGroup(elpisTrxs)
	}
	if len(wasmTrxs) > 0 {
		c.Tools().Wasm().ExecuteChainTrxsGroup(wasmTrxs)
	}
	c.appPendingTrxs = []*worker.Trx{}
}

func (c *Core) ClearAppPendingTrxs() {
	c.appPendingTrxs = []*worker.Trx{}
}

func (c *Core) SignPacket(data []byte) string {
	hashed := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privKey, cryp.SHA256, hashed[:])
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func (c *Core) ExecAppletRequestOnChain(pointId string, machineId string, key string, payload []byte, signature string, userId string, callback func([]byte, int, error)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	callbackId := crypto.SecureUniqueString()
	c.chainCallbacks[callbackId] = &chain.ChainCallback{Fn: callback, Executors: map[string]bool{}, Responses: map[string]string{}}
	var runtimeType string
	c.ModifyState(true, func(trx trx.ITrx) {
		vm := mach_model.Vm{MachineId: machineId}.Pull(trx)
		runtimeType = vm.Runtime
	})
	future.Async(func() {
		c.chain <- chain.ChainAppletRequest{Signatures: []string{c.SignPacket(payload), signature}, Submitter: c.id, RequestId: callbackId, Author: "user::" + userId, Key: key, Payload: payload, Runtime: runtimeType}
	}, false)
}

func (c *Core) ExecAppletResponseOnChain(callbackId string, packet []byte, signature string, resCode int, e string, updates []update.Update) {
	future.Async(func() {
		c.chain <- chain.ChainResponse{Signature: signature, Executor: c.id, RequestId: callbackId, ResCode: resCode, Err: e, Payload: packet, Effects: chain.Effects{DbUpdates: updates}}
	}, false)
}

func (c *Core) ExecBaseRequestOnChain(key string, payload []byte, signature string, userId string, callback func([]byte, int, error)) {
	c.lock.Lock()
	defer c.lock.Unlock()
	callbackId := crypto.SecureUniqueString()
	c.chainCallbacks[callbackId] = &chain.ChainCallback{Fn: callback, Executors: map[string]bool{}, Responses: map[string]string{}}
	for i := 0; i < 10; i++ {
		log.Println()
	}
	log.Println("test")
	for i := 0; i < 10; i++ {
		log.Println()
	}
	future.Async(func() {
		c.chain <- chain.ChainBaseRequest{Signatures: []string{c.SignPacket(payload), signature}, Submitter: c.id, RequestId: callbackId, Author: "user::" + userId, Key: key, Payload: payload}
	}, false)
}

func (c *Core) ExecBaseResponseOnChain(callbackId string, packet []byte, signature string, resCode int, e string, updates []update.Update) {
	future.Async(func() {
		c.chain <- chain.ChainResponse{Signature: signature, Executor: c.id, RequestId: callbackId, ResCode: resCode, Err: e, Payload: packet, Effects: chain.Effects{DbUpdates: updates}}
	}, false)
}

func (c *Core) OnChainPacket(typ string, trxPayload []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()
	log.Println(string(trxPayload))
	switch typ {
	case "election":
		{
			packet := chain.ChainElectionPacket{}
			err := json.Unmarshal(trxPayload, &packet)
			if err != nil {
				log.Println(err)
				return
			}
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
				if c.elections == nil {
					c.elections = []chain.Election{}
				}
				if phase == "start-reg" {
					c.elecReg = true
					c.elecStarter = voter
					c.elecStartTime = time.Now().UnixMilli()
					for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
						c.elections = append(c.elections, chain.Election{Participants: map[string]bool{}, Commits: map[string][]byte{}, Reveals: map[string]string{}})
					}
					future.Async(func() {
						c.chain <- chain.ChainElectionPacket{
							Type:    "election",
							Key:     "choose-validator",
							Meta:    map[string]any{"phase": "register", "voter": c.Ip},
							Payload: []byte("{}"),
						}
					}, false)
					if voter == c.Ip {
						future.Async(func() {
							time.Sleep(time.Duration(10) * time.Second)
							c.chain <- chain.ChainElectionPacket{
								Type:    "election",
								Key:     "choose-validator",
								Meta:    map[string]any{"phase": "end-reg", "voter": c.Ip},
								Payload: []byte("{}"),
							}
						}, false)
					}
				} else if phase == "end-reg" {
					if c.elecStarter == voter && ((time.Now().UnixMilli() - c.elecStartTime) > 8000) {
						c.elecReg = false
						payload := [][]byte{}
						nodeCount := c.babbleInst.Peers.Len()
						hasher := sha256.New()
						for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
							r := fmt.Sprintf("%d", mathrand.Intn(nodeCount))
							c.elections[i].MyNum = r
							hasher.Write([]byte(r))
							bs := hasher.Sum(nil)
							payload = append(payload, bs)
						}
						data, _ := json.Marshal(payload)
						future.Async(func() {
							c.chain <- chain.ChainElectionPacket{
								Type:    "election",
								Key:     "choose-validator",
								Meta:    map[string]any{"phase": "commit", "voter": c.Ip},
								Payload: data,
							}
						}, false)
					}
				} else if phase == "register" {
					if c.elecReg {
						for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
							c.elections[i].Participants[voter] = true
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
					if c.elections[0].Participants[voter] {
						for i := 0; i < min(MAX_VALIDATOR_COUNT, c.babbleInst.Peers.Len()); i++ {
							c.elections[i].Commits[voter] = votes[i]
						}
						if len(c.elections[0].Commits) == len(c.elections[0].Participants) {
							myReveals := []string{}
							for i := 0; i < MAX_VALIDATOR_COUNT; i++ {
								myReveals = append(myReveals, c.elections[i].MyNum)
							}
							data, _ := json.Marshal(myReveals)
							future.Async(func() {
								c.chain <- chain.ChainElectionPacket{
									Type:    "election",
									Key:     "choose-validator",
									Meta:    map[string]any{"phase": "reveal", "voter": c.Ip},
									Payload: data,
								}
							}, false)
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
					if c.elections[0].Participants[voter] {
						for i := 0; i < min(MAX_VALIDATOR_COUNT, c.babbleInst.Peers.Len()); i++ {
							c.elections[i].Reveals[voter] = votes[i]
						}
						if len(c.elections[0].Reveals) == len(c.elections[0].Participants) {
							c.executors = map[string]bool{}
							nodesArr := []string{}
							for p := range c.elections[0].Participants {
								nodesArr = append(nodesArr, p)
							}
							sort.Strings(nodesArr)
							for _, elec := range c.elections[0:min(MAX_VALIDATOR_COUNT, len(nodesArr))] {
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
								c.executors[candidate] = true
								nodesArr = append(nodesArr[:result], nodesArr[result+1:]...)
								c.elections = nil
							}
						}
					}
				}
			}
			break
		}
	case "baseRequest":
		{
			packet := chain.ChainBaseRequest{}
			err := json.Unmarshal(trxPayload, &packet)
			if err != nil {
				log.Println(err)
				return
			}
			execs := map[string]bool{}
			for k, v := range c.executors {
				execs[k] = v
			}
			if packet.Submitter == c.id {
				c.chainCallbacks[packet.RequestId].Executors = execs
			} else {
				c.chainCallbacks[packet.RequestId] = &chain.ChainCallback{Fn: nil, Executors: execs, Responses: map[string]string{}}
			}
			if !c.executors[c.Ip] {
				return
			}
			userId := ""
			if strings.HasPrefix(packet.Author, "user::") {
				userId = packet.Author[len("user::"):]
			}
			action := c.actionStore.FetchAction(packet.Key)
			if action == nil {
				return
			}
			var input input.IInput
			i, err2 := action.(iaction.ISecureAction).ParseInput("fed", string(packet.Payload))
			if err2 != nil {
				log.Println(err2)
				errText := "input parsing error"
				signature := c.SignPacket([]byte(errText))
				c.ExecBaseResponseOnChain(packet.RequestId, []byte{}, signature, 400, errText, []update.Update{})
				return
			}
			input = i
			action.(iaction.ISecureAction).SecurlyActChain(userId, packet.RequestId, packet.Payload, packet.Signatures[1], input, packet.Submitter)
			break
		}
	case "appRequest":
		{
			packet := chain.ChainAppletRequest{}
			err := json.Unmarshal(trxPayload, &packet)
			if err != nil {
				log.Println(err)
				return
			}
			execs := map[string]bool{}
			for k, v := range c.executors {
				execs[k] = v
			}
			if packet.Submitter == c.id {
				c.chainCallbacks[packet.RequestId].Executors = execs
			} else {
				c.chainCallbacks[packet.RequestId] = &chain.ChainCallback{Fn: nil, Executors: execs, Responses: map[string]string{}}
			}
			if !c.executors[c.Ip] {
				return
			}
			userId := ""
			if strings.HasPrefix(packet.Author, "user::") {
				userId = packet.Author[len("user::"):]
			}
			c.appPendingTrxs = append(c.appPendingTrxs, &worker.Trx{CallbackId: packet.RequestId, Runtime: packet.Runtime, UserId: userId, MachineId: packet.MachineId, Key: packet.Key, Payload: string(packet.Payload)})
			break
		}
	case "response":
		{
			packet := chain.ChainResponse{}
			err := json.Unmarshal(trxPayload, &packet)
			if err != nil {
				log.Println(err)
				return
			}
			callback, ok3 := c.chainCallbacks[packet.RequestId]
			if ok3 {
				if !callback.Executors[packet.Executor] {
					return
				}
				str, _ := json.Marshal(core.ResponseHolder{Payload: packet.Payload, Effects: packet.Effects})
				callback.Responses[packet.Executor] = string(str)
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
					kvstoreKeyword := "applet: "
					c.ModifyState(false, func(trx trx.ITrx) {
						for _, ef := range packet.Effects.DbUpdates {
							if (len(ef.Val) > len(kvstoreKeyword)) && (string(ef.Val[0:len(kvstoreKeyword)]) == kvstoreKeyword) {
								c.tools.Wasm().ExecuteChainEffects(string(ef.Val[len(kvstoreKeyword):]))
							} else {
								if ef.Typ == "put" {
									trx.PutBytes(ef.Key, ef.Val)
								} else if ef.Typ == "del" {
									trx.DelKey(ef.Key)
								}
							}
						}
					})
				}
				delete(c.chainCallbacks, packet.RequestId)
				if callback.Fn != nil {
					if packet.Err == "" {
						callback.Fn(packet.Payload, packet.ResCode, nil)
					} else {
						callback.Fn(packet.Payload, packet.ResCode, errors.New(packet.Err))
					}
				}
			}
			break
		}
	}
}

func (c *Core) NewHgHandler() *core.HgHandler {
	return &core.HgHandler{
		Sigma: c,
	}
}

func (c *Core) Load(gods []string, args map[string]interface{}) {
	c.gods = gods

	engine := args["babbleEngine"].(*babble.Babble)
	proxy := args["babbleProxy"].(*inmem.InmemProxy)
	sroot := args["storageRoot"].(string)
	bdbPath := args["baseDbPath"].(string)
	adbPath := args["appletDbPath"].(string)
	ldbPath := args["pointLogsDb"].(string)
	fedPort := args["federationPort"].(int)

	dnFederation := driver_network_fed.FirstStageBackFill(c)
	dstorage := driver_storage.NewStorage(c, sroot, bdbPath, ldbPath)
	dsignaler := driver_signaler.NewSignaler(c.id, dnFederation)
	dsecurity := driver_security.New(sroot, dstorage, dsignaler)
	dNetwork := driver_network.NewNetwork(c, dstorage, dsecurity, dsignaler)
	dFile := driver_file.NewFileTool(sroot)
	dDocker := driver_docker.NewDocker(c, sroot, dstorage, dFile)
	dWasm := driver_wasm.NewWasm(c, sroot, dstorage, adbPath, dDocker, dFile)
	dElpis := driver_elpis.NewElpis(c, sroot, dstorage)
	dnFederation.SecondStageForFill(fedPort, dstorage, dFile, dsignaler)

	pemData := dsecurity.FetchKeyPair("server_key")[0]
	block, _ := pem.Decode([]byte(pemData))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		panic("failed to decode PEM block containing private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	c.privKey = privateKey

	c.tools = &Tools{
		signaler: dsignaler,
		storage:  dstorage,
		security: dsecurity,
		network:  dNetwork,
		file:     dFile,
		docker:   dDocker,
		wasm:     dWasm,
		elpis:    dElpis,
	}

	c.chain = make(chan any, 1)
	future.Async(func() {
		for {
			op := <-c.chain
			typ := ""
			switch op.(type) {
			case chain.ChainBaseRequest:
				{
					typ = "baseRequest"
					break
				}
			case chain.ChainResponse:
				{
					typ = "response"
					break
				}
			case chain.ChainAppletRequest:
				{
					typ = "appRequest"
					break
				}
			case chain.ChainElectionPacket:
				{
					typ = "election"
					break
				}
			}
			if typ != "" {
				serialized, err := json.Marshal(op)
				if err == nil {
					log.Println(string(serialized))
					proxy.SubmitTx([]byte(typ + "::" + string(serialized)))
				} else {
					log.Println(err)
				}
			}
		}
	}, true)

	c.babbleInst = engine
}

func (c *Core) DoElection() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.elecReg = true
	future.Async(func() {
		c.chain <- chain.ChainElectionPacket{
			Type:    "election",
			Key:     "choose-validator",
			Meta:    map[string]any{"phase": "start-reg", "voter": c.Ip},
			Payload: []byte("{}"),
		}
	}, false)
}

func (c *Core) Run() {

	future.Async(func() {
		c.babbleInst.Run()
	}, false)

	future.Async(func() {
		for {
			time.Sleep(time.Duration(1) * time.Second)
			minutes := time.Now().Minute()
			seconds := time.Now().Second()
			if (minutes == 0) && ((seconds >= 0) && (seconds <= 2)) {
				c.DoElection()
				time.Sleep(2 * time.Minute)
			}
		}
	}, true)
}
