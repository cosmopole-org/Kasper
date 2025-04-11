package net_federation

import (
	"encoding/json"
	"errors"
	"fmt"
	"kasper/cmd/babble/sigma/abstract"
	"kasper/cmd/babble/sigma/api/model"
	outputs_invites "kasper/cmd/babble/sigma/api/outputs/invites"
	outputs_spaces "kasper/cmd/babble/sigma/api/outputs/spaces"
	updates_topics "kasper/cmd/babble/sigma/api/updates/topics"
	module_logger "kasper/cmd/babble/sigma/core/module/logger"
	"kasper/cmd/babble/sigma/layer1/adapters"
	models "kasper/cmd/babble/sigma/layer1/model"
	module_actor_model "kasper/cmd/babble/sigma/layer1/module/actor"
	"kasper/cmd/babble/sigma/layer1/tools/signaler"
	"kasper/cmd/babble/sigma/utils/crypto"
	realip "kasper/cmd/babble/sigma/utils/ip"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type FedNet struct {
	sigmaCore abstract.ICore
	storage   adapters.IStorage
	cache     adapters.ICache
	signaler  *signaler.Signaler
	logger    *module_logger.Logger
}

func FirstStageBackFill(core abstract.ICore, logger *module_logger.Logger) *FedNet {
	return &FedNet{sigmaCore: core, logger: logger}
}

func (fed *FedNet) SecondStageForFill(f *fiber.App, storage adapters.IStorage, cache adapters.ICache, signaler *signaler.Signaler) adapters.IFederation {
	fed.storage = storage
	fed.cache = cache
	fed.signaler = signaler
	f.Post("/api/federation", func(c *fiber.Ctx) error {
		var pack models.OriginPacket
		err := c.BodyParser(&pack)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.BuildErrorJson(err.Error()))
		}
		ip := realip.FromRequest(c.Context())
		hostName := ""
		for _, peer := range fed.sigmaCore.Chain().Peers.Peers {
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
				member.Id = crypto.SecureUniqueId(fed.sigmaCore.Id()) + "_" + channelId
				fed.storage.Db().Create(member)
				fed.signaler.JoinGroup(member.SpaceId, member.UserId)
			}
		}
		fed.signaler.SignalUser(payload.Key, payload.RequestId, payload.UserId, payload.Data, true)
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
			layer := fed.sigmaCore.Get(payload.Layer)
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
	for _, peer := range fed.sigmaCore.Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		addr := strings.Join(arr[0:len(arr)-1], ":")
		if addr == ipAddr {
			ok = true
			break
		}
	}
	if ok {
		statusCode, _, err := fiber.Post("https://" + destOrg + "/api/federation").JSON(packet).Bytes()
		if err != nil {
			fed.logger.Println("could not send: status: %d error: %v", statusCode, err)
		} else {
			fed.logger.Println("packet sent successfully. status: ", statusCode)
		}
	} else {
		fed.logger.Println("state org not found")
	}
}
