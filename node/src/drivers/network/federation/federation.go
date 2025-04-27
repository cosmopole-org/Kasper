package net_federation

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract/adapters/file"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	iaction "kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/models/packet"
	"kasper/src/abstract/models/trx"
	inputs_invites "kasper/src/shell/api/inputs/invites"
	inputs_points "kasper/src/shell/api/inputs/points"
	"kasper/src/shell/api/model"
	outputs_points "kasper/src/shell/api/outputs/points"
	updates_points "kasper/src/shell/api/updates/points"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type FedPacketCallback struct {
	UserId        string
	Request       []byte
	Callback      func([]byte, int, error)
	UserRequestId string
}

type FedFileCallback struct {
	Callback      *cmap.ConcurrentMap[string, func(string, error)]
	UserRequestId string
}

type FedNet struct {
	app             core.ICore
	storage         storage.IStorage
	file            file.IFile
	signaler        signaler.ISignaler
	Gateway         *Tcp
	packetCallbacks *cmap.ConcurrentMap[string, *FedPacketCallback]
	fileCallbacks   *cmap.ConcurrentMap[string, *FedFileCallback]
	fileMapLock     sync.Mutex
	Port            int
}

func FirstStageBackFill(core core.ICore) *FedNet {
	m := cmap.New[*FedPacketCallback]()
	n := cmap.New[*FedFileCallback]()
	return &FedNet{app: core, packetCallbacks: &m, fileCallbacks: &n}
}

func (fed *FedNet) SecondStageForFill(port int, storage storage.IStorage, file file.IFile, signaler signaler.ISignaler) network.IFederation {
	fed.Port = port
	fed.Gateway = NewTcp(fed.app)
	fed.storage = storage
	fed.file = file
	fed.signaler = signaler
	fed.Gateway.InjectBridge(func(socket *Socket, ip string, pack packet.OriginPacket) {
		hostName := ""
		for _, peer := range fed.app.Chain().Peers.Peers {
			arr := strings.Split(peer.NetAddr, ":")
			addr := strings.Join(arr[0:len(arr)-1], ":")
			if addr == ip {
				a, err := net.LookupAddr(ip)
				if err != nil {
					log.Println(err)
					if pack.Type == "response" {
						socket.writeResponse(pack.RequestId, 1, packet.BuildErrorJson("ip not friendly"), false)
					}
				}
				hostName = a[0]
				break
			}
		}
		log.Println("packet from ip: [", ip, "] and hostname: [", hostName, "]")
		if hostName != "" {
			fed.HandlePacket(socket, hostName, pack)
		} else {
			log.Println("hostname not known")
			if pack.Type == "response" {
				socket.writeResponse(pack.RequestId, 1, packet.BuildErrorJson("hostname not known"), false)
			}
		}
	})
	future.Async(func() {
		fed.Gateway.Listen(port)
	}, false)
	return fed
}

func ParseInput[T input.IInput](i string) (input.IInput, error) {
	body := new(T)
	err := json.Unmarshal([]byte(i), body)
	if err != nil {
		return nil, errors.New("invalid input format")
	}
	return *body, nil
}

func (fed *FedNet) HandlePacket(socket *Socket, channelId string, payload packet.OriginPacket) {
	if payload.Type == "response" {
		dataArr := strings.Split(payload.Key, " ")
		cb, ok := fed.packetCallbacks.Get(payload.RequestId)
		if ok {
			if !strings.HasPrefix(string(payload.Binary), "error: ") {
				if dataArr[0] == "/invites/accept" || dataArr[0] == "/points/join" {
					userId := ""
					pointId := ""
					if dataArr[0] == "/invites/accept" {
						var memberRes inputs_invites.AcceptInput
						err2 := json.Unmarshal(cb.Request, &memberRes)
						if err2 != nil {
							log.Println(err2)
							return
						}
						userId = cb.UserId
						pointId = memberRes.PointId
					} else if dataArr[0] == "/points/join" {
						var memberRes inputs_points.JoinInput
						err2 := json.Unmarshal(cb.Request, &memberRes)
						if err2 != nil {
							log.Println(err2)
							return
						}
						userId = cb.UserId
						pointId = memberRes.PointId
					}
					if pointId != "" {
						fed.app.ModifyState(false, func(trx trx.ITrx) {
							trx.PutLink("member::"+pointId+"::"+userId, "true")
						})
						fed.signaler.JoinGroup(pointId, userId)
					}
				} else if dataArr[0] == "/points/create" {
					var spaceOut outputs_points.CreateOutput
					err3 := json.Unmarshal(payload.Binary, &spaceOut)
					if err3 != nil {
						log.Println(err3)
						return
					}
					fed.app.ModifyState(false, func(trx trx.ITrx) {
						spaceOut.Point.Pull(trx)
						trx.PutLink("member::"+spaceOut.Point.Id+"::"+cb.UserId, "true")
					})
					fed.signaler.JoinGroup(spaceOut.Point.Id, spaceOut.Point.Id)
				}
			}
			fed.packetCallbacks.Remove(payload.RequestId)
			if strings.HasPrefix(string(payload.Binary), "error: ") {
				errPack := payload.Binary[len("error: "):]
				errObj := packet.Error{}
				json.Unmarshal([]byte(errPack), &errObj)
				err := errors.New(errObj.Message)
				cb.Callback([]byte(""), 0, err)
			} else {
				cb.Callback(payload.Binary, 1, nil)
			}
		}
	} else if payload.Type == "update" {
		reactToUpdate := func(key string, data string) {
			if key == "points/update" {
				tc := updates_points.Update{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				fed.app.ModifyState(false, func(trx trx.ITrx) {
					tc.Point.Push(trx)
				})
			} else if key == "points/delete" {
				tc := updates_points.Delete{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				fed.app.ModifyState(false, func(trx trx.ITrx) {
					trx.DelKey("obj::Point::" + tc.Point.Id)
				})
			} else if key == "points/addMember" {
				tc := updates_points.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				fed.app.ModifyState(false, func(trx trx.ITrx) {
					trx.PutLink("member::"+tc.PointId+"::"+tc.User.Id, "true")
					trx.PutLink("memberof::"+tc.User.Id+"::"+tc.PointId, "true")
				})
			} else if key == "spaces/removeMember" {
				tc := updates_points.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				fed.app.ModifyState(false, func(trx trx.ITrx) {
					trx.DelKey("link::member::" + tc.PointId + "::" + tc.User.Id)
					trx.DelKey("link::memberof::" + tc.User.Id + "::" + tc.PointId)
				})
			} else if key == "spaces/updateMember" {
				tc := updates_points.UpdateMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				fed.app.ModifyState(false, func(trx trx.ITrx) {
					trx.PutJson("member_"+tc.PointId+"_"+tc.User.Id, "meta", tc.Metadata, false)
				})
			} else if key == "points/join" {
				tc := updates_points.Join{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				fed.app.ModifyState(false, func(trx trx.ITrx) {
					trx.PutLink("member::"+tc.PointId+"::"+tc.User.Id, "true")
					trx.PutLink("memberof::"+tc.User.Id+"::"+tc.PointId, "true")
				})
			}
		}
		dataArr := strings.Split(payload.Key, " ")
		if len(dataArr) > 0 && (dataArr[0] == "update") {
			reactToUpdate(payload.Key[len("update "):], string(payload.Binary))
			fed.signaler.SignalUser(payload.Key[len("update "):], "", payload.UserId, payload.Binary, true)
		} else if len(dataArr) > 0 && (dataArr[0] == "groupUpdate") {
			reactToUpdate(payload.Key[len("groupUpdate "):], string(payload.Binary))
			fed.signaler.SignalGroup(payload.Key[len("groupUpdate "):], payload.PointId, payload.Binary, true, payload.Exceptions)
		}
	} else if payload.Type == "request" {
		action := fed.app.Actor().FetchAction(payload.Key)
		if action == nil {
			socket.writeResponse(payload.RequestId, 1, packet.BuildErrorJson("action not found"), false)
		}
		input, err := action.(iaction.ISecureAction).ParseInput("fed", payload.Binary)
		if err != nil {
			socket.writeResponse(payload.RequestId, 1, packet.BuildErrorJson("input could not be parsed"), false)
		}
		_, res, err := action.(iaction.ISecureAction).SecurelyActFed(payload.UserId, payload.Binary, payload.Signature, input)
		if err != nil {
			log.Println(err)
			socket.writeResponse(payload.RequestId, 1, packet.BuildErrorJson(err.Error()), false)
			return
		}
		socket.writeResponse(payload.RequestId, 0, res, false)
	}
}

func (fed *FedNet) SendInFederation(destOrg string, packet packet.OriginPacket) {
	ipAddr := ""
	ips, _ := net.LookupIP(destOrg)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ipAddr = ipv4.String()
			break
		}
	}
	ok := false
	for _, peer := range fed.app.Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		addr := strings.Join(arr[0:len(arr)-1], ":")
		if addr == ipAddr {
			ok = true
			break
		}
	}
	if ok {
		var statusCode int
		var err []error
		if fed.Port == 443 {
			statusCode, _, err = fiber.Post("https://" + destOrg + ":" + strconv.Itoa(fed.Port) + "/api/federation/packet").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://" + destOrg + ":" + strconv.Itoa(fed.Port) + "/api/federation/packet").JSON(packet).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: ", statusCode, " error: ", err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationPacketByCallback(destOrg string, packet packet.OriginPacket, callback func([]byte, int, error)) {
	ipAddr := ""
	ips, _ := net.LookupIP(destOrg)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ipAddr = ipv4.String()
			break
		}
	}
	ok := false
	for _, peer := range fed.app.Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		addr := strings.Join(arr[0:len(arr)-1], ":")
		if addr == ipAddr {
			ok = true
			break
		}
	}
	if ok {
		callbackId := crypto.SecureUniqueString()
		cb := &FedPacketCallback{Callback: callback, UserRequestId: packet.RequestId, Request: packet.Binary, UserId: packet.UserId}
		packet.RequestId = callbackId
		fed.packetCallbacks.Set(callbackId, cb)
		future.Async(func() {
			time.Sleep(time.Duration(120) * time.Second)
			cb, ok := fed.packetCallbacks.Get(callbackId)
			if ok {
				fed.packetCallbacks.Remove(callbackId)
				cb.Callback([]byte(""), 0, errors.New("federation callback timeout"))
			}
		}, false)
		var statusCode int
		var err []error
		if fed.Port == 443 {
			statusCode, _, err = fiber.Post("https://" + destOrg + ":" + strconv.Itoa(fed.Port) + "/api/federation/packet").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://" + destOrg + ":" + strconv.Itoa(fed.Port) + "/api/federation/packet").JSON(packet).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: ", statusCode, " error: ", err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationFileReqByCallback(destOrg string, fileId string, packet packet.OriginPacket, callback func(string, error)) {
	ipAddr := ""
	ips, _ := net.LookupIP(destOrg)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ipAddr = ipv4.String()
			break
		}
	}
	ok := false
	for _, peer := range fed.app.Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		addr := strings.Join(arr[0:len(arr)-1], ":")
		if addr == ipAddr {
			ok = true
			break
		}
	}
	if ok {
		fileObj := model.File{Id: fileId}
		found := false
		fed.app.ModifyState(true, func(trx trx.ITrx) {
			found = trx.HasObj("File", fileId)
			fileObj.Pull(trx)
		})
		if found {
			path := fmt.Sprintf("%s/files/%s/%s", fed.storage.StorageRoot(), packet.PointId, fileId)
			callback(path, nil)
			return
		}
		callbackId := crypto.SecureUniqueString()
		func() {
			fed.fileMapLock.Lock()
			defer fed.fileMapLock.Unlock()
			cb, ok := fed.fileCallbacks.Get(fileId)
			if ok {
				cb.Callback.Set(callbackId, callback)
			} else {
				m := cmap.New[func(string, error)]()
				m.Set(callbackId, callback)
				cb = &FedFileCallback{Callback: &m}
				fed.fileCallbacks.Set(fileId, cb)
			}
		}()
		future.Async(func() {
			time.Sleep(time.Duration(120) * time.Second)
			cb, ok := fed.fileCallbacks.Get(fileId)
			if ok {
				cbSingle, ok := cb.Callback.Get(callbackId)
				if ok {
					cb.Callback.Remove(callbackId)
					cbSingle("", errors.New("federation callback timeout"))
				}
			}
		}, false)
		var statusCode int
		var err []error
		if fed.Port == 443 {
			statusCode, _, err = fiber.Get("https://" + destOrg + ":" + strconv.Itoa(fed.Port) + "/api/federation/getFile").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Get("http://" + destOrg + ":" + strconv.Itoa(fed.Port) + "/api/federation/getFile").JSON(packet).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: ", statusCode, " error: ", err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationFileResByCallback(destOrg string, pack packet.OriginFileRes) {
	ipAddr := ""
	ips, _ := net.LookupIP(destOrg)
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ipAddr = ipv4.String()
			break
		}
	}
	ok := false
	for _, peer := range fed.app.Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		addr := strings.Join(arr[0:len(arr)-1], ":")
		if addr == ipAddr {
			ok = true
			break
		}
	}
	if ok {
		var file = model.File{Id: pack.FileId}
		fed.app.ModifyState(true, func(trx trx.ITrx) {
			file.Pull(trx)
		})
		fileInfo, _ := json.Marshal(file)
		var statusCode int
		var err []error
		args := fiber.AcquireArgs()
		defer fiber.ReleaseArgs(args)
		args.Set("FileInfo", string(fileInfo))
		args.Set("UserId", pack.UserId)
		args.Set("PointId", pack.PointId)
		args.Set("RequestId", pack.RequestId)
		path := fmt.Sprintf("%s/files/%s/%s", fed.storage.StorageRoot(), pack.PointId, pack.FileId)
		if fed.Port == 443 {
			statusCode, _, err = fiber.Post("https://"+destOrg+":"+strconv.Itoa(fed.Port)+"/api/federation/putFile").SendFile(path, "Data").MultipartForm(args).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://"+destOrg+":"+strconv.Itoa(fed.Port)+"/api/federation/putFile").SendFile(path, "Data").MultipartForm(args).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: ", statusCode, " error: ", err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}
