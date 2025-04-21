package signaler

import (
	"encoding/json"
	"kasper/src/abstract/adapters/network"
	"kasper/src/abstract/adapters/signaler"
	module_logger "kasper/src/core/module/logger"
	models "kasper/src/shell/layer1/model"
	"strings"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
)

const groupUpdatePrefix = "groupUpdate "
const updatePrefix = "update "
const responsePrefix = "response "

type Signaler struct {
	Lock           sync.Mutex
	appId          string
	listeners      *cmap.ConcurrentMap[string, *signaler.Listener]
	groups         *cmap.ConcurrentMap[string, *signaler.Group]
	GlobalBridge   *signaler.GlobalListener
	LGroupDisabled bool
	JListener      *signaler.JoinListener
	Federation     network.IFederation
	logger         *module_logger.Logger
}

func (p *Signaler) Listeners() *cmap.ConcurrentMap[string, *signaler.Listener] {
	return p.listeners
}

func (p *Signaler) Groups() *cmap.ConcurrentMap[string, *signaler.Group] {
	return p.groups
}

func (p *Signaler) ListenToSingle(listener *signaler.Listener) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.listeners.Set(listener.Id, listener)
}

func (p *Signaler) ListenToGroup(listener *signaler.Listener, overrideFunctionaly bool) {
	g, _ := p.RetriveGroup(listener.Id)
	g.Listener = listener
	g.Override = overrideFunctionaly
}

func (p *Signaler) BrdigeGlobally(listener *signaler.GlobalListener, overrideFunctionaly bool) {
	p.LGroupDisabled = true
	p.GlobalBridge = listener
}

func (p *Signaler) ListenToJoin(listener *signaler.JoinListener) {
	p.JListener = listener
}

func (p *Signaler) SignalUser(key string, respondToId string, listenerId string, data any, pack bool) {
	if !strings.Contains(listenerId, "@") {
		return
	}
	var isVm = false
	if listenerId[0:2] == "b_" {
		isVm = true
	}
	origin := strings.Split(listenerId, "@")[1]
	if origin == p.appId || isVm || origin == "global" {
		listener, found := p.listeners.Get(listenerId)
		if found && (listener != nil) {
			if pack {
				var message string
				switch d := data.(type) {
				case string:
					message = d
				default:
					msg, err := json.Marshal(d)
					if err != nil {
						p.logger.Println(err)
						return
					}
					message = string(msg)
				}
				if len(respondToId) > 0 {
					listener.Signal([]byte(responsePrefix + respondToId + " " + message))
				} else {
					listener.Signal([]byte(updatePrefix + key + " " + message))
				}
			} else {
				listener.Signal(data)
			}
		}
	} else {
		var message string
		switch d := data.(type) {
		case string:
			message = d
		default:
			msg, err := json.Marshal(d)
			if err != nil {
				p.logger.Println(err)
				return
			}
			message = string(msg)
		}
		p.Federation.SendInFederation(origin, models.OriginPacket{IsResponse: false, Key: updatePrefix + key, UserId: listenerId, Data: message})
	}
}

func (p *Signaler) SignalGroup(key string, groupId string, data any, pack bool, exceptions []string) {
	var excepDict = map[string]bool{}
	for _, exc := range exceptions {
		excepDict[exc] = true
	}
	group, ok := p.RetriveGroup(groupId)
	if ok {
		var packet any
		if pack {
			var message []byte
			switch d := data.(type) {
			case string:
				message = []byte(d)
			default:
				msg, err := json.Marshal(d)
				if err != nil {
					p.logger.Println(err)
					return
				}
				message = msg
			}
			packet = []byte(updatePrefix + key + " " + string(message))
		} else {
			packet = data
		}
		if p.LGroupDisabled {
			p.GlobalBridge.Signal(groupId, packet)
			return
		}
		if group.Override {
			group.Listener.Signal(packet)
			return
		}
		var foreignersMap = map[string][]string{}
		for t := range group.Points.IterBuffered() {
			userId := t.Val
			if !strings.Contains(userId, "@") {
				continue
			}
			var isVm = false
			if userId[0:2] == "b_" {
				isVm = true
			}
			userOrigin := strings.Split(userId, "@")[1]
			if (userOrigin == p.appId) || isVm || (userOrigin == "global") {
				if !p.LGroupDisabled || !group.Override {
					if !excepDict[t.Key] {
						listener, found := p.listeners.Get(userId)
						if found && (listener != nil) {
							listener.Signal(packet)
						}
					}
				}
			} else {
				if foreignersMap[userOrigin] == nil {
					foreignersMap[userOrigin] = []string{}
				}
				if excepDict[t.Key] {
					foreignersMap[userOrigin] = append(foreignersMap[userOrigin], t.Val)
				}
			}
		}
		var message string
		switch d := data.(type) {
		case string:
			message = d
		default:
			msg, err := json.Marshal(d)
			if err != nil {
				p.logger.Println(err)
				return
			}
			message = string(msg)
		}
		for k, v := range foreignersMap {
			p.Federation.SendInFederation(k, models.OriginPacket{IsResponse: false, Key: groupUpdatePrefix + key, SpaceId: groupId, Exceptions: v, Data: message})
		}
	}
}

func (p *Signaler) JoinGroup(groupId string, userId string) {
	g, ok := p.RetriveGroup(groupId)
	if ok {
		g.Points.Set(userId, userId)
		if p.JListener != nil {
			p.JListener.Join(groupId, userId)
		}
	}
}

func (p *Signaler) LeaveGroup(groupId string, userId string) {
	g, ok := p.RetriveGroup(groupId)
	if ok {
		g.Points.Remove(userId)
		if p.JListener != nil {
			p.JListener.Leave(groupId, userId)
		}
	}
}

func (p *Signaler) RetriveGroup(groupId string) (*signaler.Group, bool) {
	ok := p.groups.Has(groupId)
	if !ok {
		newMap := cmap.New[string]()
		group := &signaler.Group{Points: &newMap, Listener: nil, Override: false}
		p.groups.SetIfAbsent(groupId, group)
	}
	return p.groups.Get(groupId)
}

func NewSignaler(appId string, logger *module_logger.Logger, federation network.IFederation) *Signaler {
	logger.Println("creating signaler...")
	newMap := cmap.New[*signaler.Group]()
	lisMap := cmap.New[*signaler.Listener]()
	return &Signaler{
		appId:          appId,
		logger:         logger,
		listeners:      &lisMap,
		groups:         &newMap,
		LGroupDisabled: false,
		Federation:     federation,
		GlobalBridge:   nil,
	}
}
