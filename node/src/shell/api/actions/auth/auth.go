package actions_auth

import (
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/state"
	inputsauth "kasper/src/shell/api/inputs/auth"
	outputsauth "kasper/src/shell/api/outputs/auth"
	"strings"
)

type Actions struct {
	app core.ICore
}

func Install(a *Actions) error {
	return nil
}

// GetServerPublicKey /auths/getServerPublicKey check [ false false false ] access [ true false false false GET ]
func (a *Actions) GetServerPublicKey(_ state.IState, _ inputsauth.GetServerKeyInput) (any, error) {
	return &outputsauth.GetServerKeyOutput{PublicKey: string(a.app.Tools().Security().FetchKeyPair("server_key")[1])}, nil
}

// GetServersMap /auths/getServersMap check [ false false false ] access [ true false false false GET ]
func (a *Actions) GetServersMap(_ state.IState, _ inputsauth.GetServersMapInput) (any, error) {
	m := []string{}
	for _, peer := range a.app.Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		m = append(m, strings.Join(arr[0:len(arr)-1], ":"))
	}
	return outputsauth.GetServersMapOutput{Servers: m}, nil
}
