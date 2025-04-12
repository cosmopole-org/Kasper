package net_federation

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/src/abstract"
	module_logger "kasper/src/core/module/logger"
	"kasper/src/shell/api/model"
	outputs_invites "kasper/src/shell/api/outputs/invites"
	outputs_spaces "kasper/src/shell/api/outputs/spaces"
	updates_spaces "kasper/src/shell/api/updates/spaces"
	updates_topics "kasper/src/shell/api/updates/topics"
	"kasper/src/shell/layer1/adapters"
	models "kasper/src/shell/layer1/model"
	module_actor_model "kasper/src/shell/layer1/module/actor"
	"kasper/src/shell/layer1/tools/signaler"
	net_http "kasper/src/shell/layer3/tools/network/http"
	"kasper/src/shell/utils/crypto"
	"kasper/src/shell/utils/future"
	realip "kasper/src/shell/utils/ip"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type FedCallback struct {
	Callback      func([]byte, int, error)
	UserRequestId string
}

type FedNet struct {
	app        abstract.ICore
	storage    adapters.IStorage
	cache      adapters.ICache
	signaler   *signaler.Signaler
	logger     *module_logger.Logger
	httpServer *net_http.HttpServer
	callbacks  *cmap.ConcurrentMap[string, FedCallback]
}

func FirstStageBackFill(core abstract.ICore, logger *module_logger.Logger) *FedNet {
	m := cmap.New[FedCallback]()
	return &FedNet{app: core, logger: logger, callbacks: &m}
}

func (fed *FedNet) SecondStageForFill(f *net_http.HttpServer, storage adapters.IStorage, cache adapters.ICache, signaler *signaler.Signaler) adapters.IFederation {
	fed.httpServer = f
	fed.storage = storage
	fed.cache = cache
	fed.signaler = signaler
	fed.httpServer.Server.Post("/api/federation", func(c *fiber.Ctx) error {
		var pack models.OriginPacket
		err := c.BodyParser(&pack)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.BuildErrorJson(err.Error()))
		}
		ip := realip.FromRequest(c.Context())
		hostName := ""
		for _, peer := range fed.app.Chain().Peers.Peers {
			arr := strings.Split(peer.NetAddr, ":")
			addr := strings.Join(arr[0:len(arr)-1], ":")
			if addr == ip {
				a, err := net.LookupAddr(ip)
				if err != nil {
					fed.logger.Println(err)
					return errors.New("ip not friendly")
				}
				hostName = a[0]
				break
			}
		}
		fed.logger.Println("packet from ip: [", ip, "] and hostname: [", hostName, "]")
		if hostName != "" {
			fed.HandlePacket(hostName, pack)
			return c.Status(fiber.StatusOK).JSON(models.ResponseSimpleMessage{Message: "federation packet received"})
		} else {
			fed.logger.Println("hostname not known")
			return c.Status(fiber.StatusOK).JSON(models.ResponseSimpleMessage{Message: "hostname not known"})
		}
	})
	return fed
}

func ParseInput[T abstract.IInput](i string) (abstract.IInput, error) {
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
		if dataArr[0] == "/invites/accept" || dataArr[0] == "/spaces/join" {
			var member *model.Member
			if dataArr[0] == "/invites/accept" {
				var memberRes outputs_invites.AcceptOutput
				err2 := json.Unmarshal([]byte(payload.Data), &memberRes)
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
				member = &memberRes.Member
			} else if dataArr[0] == "/spaces/join" {
				var memberRes outputs_spaces.JoinOutput
				err2 := json.Unmarshal([]byte(payload.Data), &memberRes)
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
				member = &memberRes.Member
			}
			if member != nil {
				member.Id = crypto.SecureUniqueId(fed.app.Id()) + "_" + channelId
				fed.storage.Db().Create(member)
				fed.signaler.JoinGroup(member.SpaceId, member.UserId)
			}
		} else if dataArr[0] == "/spaces/create" {
			var spaceRes outputs_spaces.CreateOutput
			err2 := json.Unmarshal([]byte(payload.Data), &spaceRes)
			if err2 != nil {
				fed.logger.Println(err2)
				return
			}
			fed.storage.DoTrx(func(trx adapters.ITrx) error {
				trx.Db().Create(&spaceRes.Space)
				trx.Db().Create(&spaceRes.Topic)
				trx.Db().Create(&spaceRes.Member)
				return nil
			})
			fed.cache.Put(fmt.Sprintf("city::%s", spaceRes.Topic.Id), spaceRes.Topic.SpaceId)
			fed.signaler.JoinGroup(spaceRes.Member.SpaceId, spaceRes.Member.UserId)
			fed.cache.Put(fmt.Sprintf(memberTemplate, spaceRes.Member.SpaceId, spaceRes.Member.UserId, spaceRes.Member.Id), spaceRes.Member.TopicId)
		}
		cb, ok := fed.callbacks.Get(payload.RequestId)
		if ok {
			fed.callbacks.Remove(payload.RequestId)
			if strings.HasPrefix(payload.Data, "error: ") {
				errPack := payload.Data[len("error: "):]
				errObj := models.Error{}
				json.Unmarshal([]byte(errPack), &errObj)
				err := errors.New(errObj.Message)
				cb.Callback([]byte(""), 0, err)
			} else {
				cb.Callback([]byte(payload.Data), 1, nil)
			}
		}
	} else {
		reactToUpdate := func(key string, data string) {
			if key == "topics/create" {
				tc := updates_topics.Create{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Create(&tc.Topic).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
				fed.cache.Put(fmt.Sprintf("city::%s", tc.Topic.Id), tc.Topic.SpaceId)
			} else if key == "topics/update" {
				tc := updates_topics.Update{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Save(&tc.Topic).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "topics/delete" {
				tc := updates_topics.Delete{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Delete(&tc.Topic).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "spaces/update" {
				tc := updates_spaces.Update{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Save(&tc.Space).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "spaces/delete" {
				tc := updates_spaces.Delete{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Delete(&tc.Space).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "spaces/addMember" {
				tc := updates_spaces.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Create(&tc.Member).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "spaces/removeMember" {
				tc := updates_spaces.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Delete(&tc.Member).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "spaces/updateMember" {
				tc := updates_spaces.AddMember{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Save(&tc.Member).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
					return
				}
			} else if key == "spaces/join" {
				tc := updates_spaces.Join{}
				err := json.Unmarshal([]byte(data), &tc)
				if err != nil {
					fed.logger.Println(err)
					return
				}
				err2 := fed.storage.DoTrx(func(trx adapters.ITrx) error {
					return trx.Db().Create(&tc.Member).Error
				})
				if err2 != nil {
					fed.logger.Println(err2)
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
				fed.logger.Println(err)
				errPack, err2 := json.Marshal(models.BuildErrorJson(err.Error()))
				if err2 == nil {
					errPack = []byte("error: " + string(errPack))
					fed.SendInFederation(channelId, models.OriginPacket{IsResponse: true, Key: payload.Key, RequestId: payload.RequestId, Data: string(errPack), UserId: payload.UserId})
				}
				return
			}
			packet, err3 := json.Marshal(res)
			if err3 != nil {
				fed.logger.Println(err3)
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
		if fed.httpServer.Port == 443 {
			statusCode, _, err = fiber.Post("https://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port) + "/api/federation").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port) + "/api/federation").JSON(packet).Bytes()
		}
		if err != nil {
			fed.logger.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			fed.logger.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		fed.logger.Println("state org not found")
	}
}

func (fed *FedNet) SendInFederationByCallback(destOrg string, packet models.OriginPacket, callback func([]byte, int, error)) {
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
		cb := FedCallback{Callback: callback, UserRequestId: packet.RequestId}
		packet.RequestId = callbackId
		fed.callbacks.Set(callbackId, cb)
		future.Async(func() {
			time.Sleep(time.Duration(120) * time.Second)
			cb, ok := fed.callbacks.Get(callbackId)
			if ok {
				fed.callbacks.Remove(callbackId)
				cb.Callback([]byte(""), 0, errors.New("federation callback timeout"))
			}
		}, false)
		var statusCode int
		var err []error
		if fed.httpServer.Port == 443 {
			statusCode, _, err = fiber.Post("https://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port) + "/api/federation").JSON(packet).Bytes()
		} else {
			statusCode, _, err = fiber.Post("http://" + destOrg + ":" + strconv.Itoa(fed.httpServer.Port) + "/api/federation").JSON(packet).Bytes()
		}
		if err != nil {
			fed.logger.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			fed.logger.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		fed.logger.Println("state org not found")
	}
}
