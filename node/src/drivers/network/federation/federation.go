package net_federation

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract/adapters/file"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/signaler"
	"kasper/src/abstract/adapters/storage"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	"kasper/src/abstract/models/trx"
	inputs_invites "kasper/src/shell/api/inputs/invites"
	inputs_spaces "kasper/src/shell/api/inputs/spaces"
	inputs_storage "kasper/src/shell/api/inputs/storage"
	"kasper/src/shell/api/model"
	outputs_invites "kasper/src/shell/api/outputs/invites"
	outputs_spaces "kasper/src/shell/api/outputs/spaces"
	updates_spaces "kasper/src/shell/api/updates/spaces"
	updates_topics "kasper/src/shell/api/updates/topics"
	models "kasper/src/shell/layer1/model"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	realip "kasper/src/shell/utils/ip"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mitchellh/mapstructure"
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
	httpServer      network.IHttp
	packetCallbacks *cmap.ConcurrentMap[string, *FedPacketCallback]
	fileCallbacks   *cmap.ConcurrentMap[string, *FedFileCallback]
	fileMapLock     sync.Mutex
}

func FirstStageBackFill(core core.ICore) *FedNet {
	m := cmap.New[*FedPacketCallback]()
	n := cmap.New[*FedFileCallback]()
	return &FedNet{app: core, packetCallbacks: &m, fileCallbacks: &n}
}

func (fed *FedNet) SecondStageForFill(f network.IHttp, storage storage.IStorage, file file.IFile, signaler signaler.ISignaler) network.IFederation {
	fed.httpServer = f
	fed.storage = storage
	fed.file = file
	fed.signaler = signaler
	fed.httpServer.Server().Post("/api/federation/packet", func(c *fiber.Ctx) error {
		ip := realip.FromRequest(c.Context())
		hostName := ""
		for _, peer := range fed.app.Chain().Peers.Peers {
			arr := strings.Split(peer.NetAddr, ":")
			addr := strings.Join(arr[0:len(arr)-1], ":")
			if addr == ip {
				a, err := net.LookupAddr(ip)
				if err != nil {
					log.Println(err)
					return errors.New("ip not friendly")
				}
				hostName = a[0]
				break
			}
		}
		log.Println("packet from ip: [", ip, "] and hostname: [", hostName, "]")
		if hostName != "" {
			var pack models.OriginPacket
			err := c.BodyParser(&pack)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(models.BuildErrorJson(err.Error()))
			}
			fed.HandlePacket(hostName, pack)
			return c.Status(fiber.StatusOK).JSON(models.ResponseSimpleMessage{Message: "federation packet received"})
		} else {
			log.Println("hostname not known")
			return c.Status(fiber.StatusInternalServerError).JSON(models.ResponseSimpleMessage{Message: "hostname not known"})
		}
	})
	fed.httpServer.Server().Post("/api/federation/putFile", func(c *fiber.Ctx) error {
		ip := realip.FromRequest(c.Context())
		hostName := ""
		for _, peer := range fed.app.Chain().Peers.Peers {
			arr := strings.Split(peer.NetAddr, ":")
			addr := strings.Join(arr[0:len(arr)-1], ":")
			if addr == ip {
				a, err := net.LookupAddr(ip)
				if err != nil {
					log.Println(err)
					return errors.New("ip not friendly")
				}
				hostName = a[0]
				break
			}
		}
		log.Println("packet from ip: [", ip, "] and hostname: [", hostName, "]")
		if hostName != "" {
			data := new(models.OriginFile)
			form, err := c.MultipartForm()
			if err == nil {
				var formData = map[string]any{}
				for k, v := range form.Value {
					formData[k] = v[0]
				}
				for k, v := range form.File {
					formData[k] = v[0]
				}
				err := mapstructure.Decode(formData, data)
				if err != nil {
					return err
				}
				file := model.File{}
				e := json.Unmarshal([]byte(data.FileInfo), &file)
				if e != nil {
					log.Println(e)
					return e
				}
				if fed.fileCallbacks.Has(file.Id) {
					path := fmt.Sprintf("%s/files/%s/%s", fed.storage.StorageRoot(), data.TopicId, file.Id)
					fed.file.SaveFileToStorage(fed.storage.StorageRoot(), data.Data, data.TopicId, file.Id)
					fed.app.ModifyState(false, func(trx trx.ITrx) {
						file.Push(trx)
					})
					var cb *FedFileCallback
					var ok bool
					func() {
						fed.fileMapLock.Lock()
						defer fed.fileMapLock.Unlock()
						cb, ok = fed.fileCallbacks.Get(file.Id)
						if ok {
							fed.fileCallbacks.Remove(file.Id)
						}
					}()
					if ok {
						for _, callback := range cb.Callback.Items() {
							callback(path, nil)
						}
					}
				}
				return c.Status(fiber.StatusOK).JSON(map[string]any{})
			} else {
				return err
			}
		} else {
			log.Println("hostname not known")
			return c.Status(fiber.StatusInternalServerError).JSON(models.ResponseSimpleMessage{Message: "hostname not known"})
		}
	})
	fed.httpServer.Server().Post("/api/federation/getFile", func(c *fiber.Ctx) error {
		ip := realip.FromRequest(c.Context())
		hostName := ""
		for _, peer := range fed.app.Chain().Peers.Peers {
			arr := strings.Split(peer.NetAddr, ":")
			addr := strings.Join(arr[0:len(arr)-1], ":")
			if addr == ip {
				a, err := net.LookupAddr(ip)
				if err != nil {
					log.Println(err)
					return errors.New("ip not friendly")
				}
				hostName = a[0]
				break
			}
		}
		log.Println("packet from ip: [", ip, "] and hostname: [", hostName, "]")
		if hostName != "" {
			data := new(models.OriginPacket)
			err := c.BodyParser(data)
			if err != nil {
				return err
			}
			inp := inputs_storage.DownloadInput{}
			var trxErr error = nil
			fed.app.ModifyState(true, func(trx trx.ITrx) {
				json.Unmarshal(data.Binary, &inp)
				if !trx.HasObj("File", inp.FileId) {
					trxErr = errors.New("file not found")
					return
				}
				var file = model.File{Id: inp.FileId}.Pull(trx)
				if file.PointId != data.PointId {
					trxErr = errors.New("access to file denied")
				}
			})
			if trxErr != nil {
				log.Println(trxErr)
				return c.Status(fiber.StatusInternalServerError).JSON(models.ResponseSimpleMessage{Message: trxErr.Error()})
			}
			fed.SendInFederationFileResByCallback(ip, models.OriginFileRes{
				UserId:    data.UserId,
				PointId:   data.PointId,
				RequestId: data.RequestId,
				FileId:    inp.FileId,
			})
			return c.Status(fiber.StatusOK).JSON(models.ResponseSimpleMessage{Message: "federation packet received"})
		} else {
			log.Println("hostname not known")
			return c.Status(fiber.StatusInternalServerError).JSON(models.ResponseSimpleMessage{Message: "hostname not known"})
		}
	})
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

const memberTemplate = "member::%s::%s::%s"

func (fed *FedNet) HandlePacket(channelId string, payload models.OriginPacket) {
	if payload.IsResponse {
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
						parts := strings.Split(memberRes.InviteId, "::")
						userId = parts[2]
						pointId = parts[3]
					} else if dataArr[0] == "/points/join" {
						var memberRes inputs_spaces.JoinInput
						err2 := json.Unmarshal(cb.Request, &memberRes)
						if err2 != nil {
							log.Println(err2)
							return
						}
						userId = cb.UserId
						pointId = memberRes.SpaceId
					}
					if pointId != "" {
						fed.app.ModifyState(false, func(trx trx.ITrx) {
							trx.PutLink("member::" + pointId + "::" + userId, "true")
						})
						fed.signaler.JoinGroup(pointId, userId)
					}
				} else if dataArr[0] == "/points/create" {
					var spaceOut outputs_spaces.CreateOutput
					err3 := json.Unmarshal(payload.Binary, &spaceOut)
					if err3 != nil {
						log.Println(err3)
						return
					}
				    fed.app.ModifyState(false, func(trx trx.ITrx) {
						spaceOut.Point.Pull(trx)
						trx.PutLink("member::" + spaceOut.Point.Id + "::" + cb.UserId, "true")
					})
					fed.signaler.JoinGroup(spaceOut.Point.Id, spaceOut.Point.Id)
				}
			}
			fed.packetCallbacks.Remove(payload.RequestId)
			if strings.HasPrefix(string(payload.Binary), "error: ") {
				errPack := payload.Binary[len("error: "):]
				errObj := models.Error{}
				json.Unmarshal([]byte(errPack), &errObj)
				err := errors.New(errObj.Message)
				cb.Callback([]byte(""), 0, err)
			} else {
				cb.Callback(payload.Binary, 1, nil)
			}
		}
	} else {
		reactToUpdate := func(key string, data string) {
			if key == "topics/create" {
				tc := updates_topics.Create{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Create(&tc.Topic).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
				fed.cache.Put(fmt.Sprintf("city::%s", tc.Topic.Id), tc.Topic.SpaceId)
			} else if key == "topics/update" {
				tc := updates_topics.Update{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Save(&tc.Topic).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "topics/delete" {
				tc := updates_topics.Delete{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Delete(&tc.Topic).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "spaces/update" {
				tc := updates_spaces.Update{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Save(&tc.Space).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "spaces/delete" {
				tc := updates_spaces.Delete{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Delete(&tc.Space).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "spaces/addMember" {
				tc := updates_spaces.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Create(&tc.Member).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "spaces/removeMember" {
				tc := updates_spaces.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Delete(&tc.Member).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "spaces/updateMember" {
				tc := updates_spaces.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Save(&tc.Member).Error
				})
				if err2 != nil {
					log.Println(err2)
					return
				}
			} else if key == "spaces/join" {
				tc := updates_spaces.Join{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					log.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Create(&tc.Member).Error
				})
				if err2 != nil {
					log.Println(err2)
					// nevermin if there is an error about duplication
				}
			}
		}
		dataArr := strings.Split(payload.Key, " ")
		if len(dataArr) > 0 && (dataArr[0] == "update") {
			reactToUpdate(payload.Key[len("update "):], payload.Data)
			fed.signaler.SignalUser(payload.Key[len("update "):], "", payload.UserId, payload.Data, true)
		} else if len(dataArr) > 0 && (dataArr[0] == "groupUpdate") {
			reactToUpdate(payload.Key[len("groupUpdate "):], payload.Data)
			fed.signaler.SignalGroup(payload.Key[len("groupUpdate "):], payload.SpaceId, payload.Data, true, payload.Exceptions)
		} else {
			layer := fed.app.Get(payload.Layer)
			if layer == nil {
				return
			}
			action := layer.Actor().FetchAction(payload.Key)
			if action == nil {
				errPack, _ := json.Marshal(models.BuildErrorJson("action not found"))
				fed.SendInFederation(channelId, models.OriginPacket{IsResponse: true, Key: payload.Key, RequestId: payload.RequestId, Data: string(errPack), UserId: payload.UserId})
			}
			input, err := action.(*module_actor_model.SecureAction).ParseInput("fed", payload.Data)
			if err != nil {
				errPack, _ := json.Marshal(models.BuildErrorJson("input could not be parsed"))
				fed.SendInFederation(channelId, models.OriginPacket{IsResponse: true, Key: payload.Key, RequestId: payload.RequestId, Data: string(errPack), UserId: payload.UserId})
			}
			_, res, err := action.(*module_actor_model.SecureAction).SecurelyActFed(layer, payload.UserId, input)
			if err != nil {
				log.Println(err)
				errPack, err2 := json.Marshal(models.BuildErrorJson(err.Error()))
				if err2 == nil {
					errPack = []byte("error: " + string(errPack))
					fed.SendInFederation(channelId, models.OriginPacket{IsResponse: true, Key: payload.Key, RequestId: payload.RequestId, Data: string(errPack), UserId: payload.UserId})
				}
				return
			}
			packet, err3 := json.Marshal(res)
			if err3 != nil {
				log.Println(err3)
				errPack, err2 := json.Marshal(models.BuildErrorJson(err3.Error()))
				if err2 == nil {
					fed.SendInFederation(channelId, models.OriginPacket{IsResponse: true, Key: payload.Key, RequestId: payload.RequestId, Data: string(errPack), UserId: payload.UserId})
				}
				return
			}
			fed.SendInFederation(channelId, models.OriginPacket{IsResponse: true, Key: payload.Key, RequestId: payload.RequestId, Data: string(packet), UserId: payload.UserId})
		}
	}
}

func (fed *FedNet) SendInFederation(destOrg string, packet models.OriginPacket) {
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
		if fed.httpServer.Port() == 443 {
			statusCode, _, err = fiber.Post("https://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port()) + "/api/federation/packet").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port()) + "/api/federation/packet").JSON(packet).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationPacketByCallback(destOrg string, packet models.OriginPacket, callback func([]byte, int, error)) {
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
		if fed.httpServer.Port() == 443 {
			statusCode, _, err = fiber.Post("https://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port()) + "/api/federation/packet").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port()) + "/api/federation/packet").JSON(packet).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationFileReqByCallback(destOrg string, fileId string, packet models.OriginPacket, callback func(string, error)) {
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
		if fed.httpServer.Port() == 443 {
			statusCode, _, err = fiber.Get("https://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port()) + "/api/federation/getFile").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Get("http://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port()) + "/api/federation/getFile").JSON(packet).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationFileResByCallback(destOrg string, packet models.OriginFileRes) {
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
		var file = model.File{Id: packet.FileId}
		fed.app.ModifyState(true, func(trx trx.ITrx) {
			file.Pull(trx)
		})
		fileInfo, _ := json.Marshal(file)
		var statusCode int
		var err []error
		args := fiber.AcquireArgs()
		defer fiber.ReleaseArgs(args)
		args.Set("FileInfo", string(fileInfo))
		args.Set("UserId", packet.UserId)
		args.Set("PointId", packet.PointId)
		args.Set("RequestId", packet.RequestId)
		path := fmt.Sprintf("%s/files/%s/%s", fed.storage.StorageRoot(), packet.PointId, packet.FileId)
		if fed.httpServer.Port() == 443 {
			statusCode, _, err = fiber.Post("https://"+destOrg+":"+strconv.Itoa(fed.httpServer.Port())+"/api/federation/putFile").SendFile(path, "Data").MultipartForm(args).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://"+destOrg+":"+strconv.Itoa(fed.httpServer.Port())+"/api/federation/putFile").SendFile(path, "Data").MultipartForm(args).Bytes()
		}
		if err != nil {
			log.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			log.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		log.Println("state org not found")
	}
}
