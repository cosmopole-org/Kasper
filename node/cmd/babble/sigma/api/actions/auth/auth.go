package actions_auth

import (
	"kasper/cmd/babble/sigma/abstract"
	inputsauth "kasper/cmd/babble/sigma/api/inputs/auth"
	outputsauth "kasper/cmd/babble/sigma/api/outputs/auth"
	"kasper/cmd/babble/sigma/layer1/adapters"
	tb "kasper/cmd/babble/sigma/layer1/module/toolbox"
	"strings"
)

type Actions struct {
	Layer abstract.ILayer
}

func Install(s adapters.IStorage, a *Actions) error {
	return nil
}

// GetServerPublicKey /auths/getServerPublicKey check [ false false false ] access [ true false false false GET ]
func (a *Actions) GetServerPublicKey(_ abstract.IState, _ inputsauth.GetServerKeyInput) (any, error) {
	toolbox := abstract.UseToolbox[*tb.ToolboxL1](a.Layer.Tools())
	return &outputsauth.GetServerKeyOutput{PublicKey: string(toolbox.Security().FetchKeyPair("server_key")[1])}, nil
}

// GetServersMap /auths/getServersMap check [ false false false ] access [ true false false false GET ]
func (a *Actions) GetServersMap(_ abstract.IState, _ inputsauth.GetServersMapInput) (any, error) {
	m := []string{}
	for _, peer := range a.Layer.Core().Chain().Peers.Peers {
		arr := strings.Split(peer.NetAddr, ":")
		m = append(m, strings.Join(arr[0 : len(arr)-1], ":"))
	}
	return outputsauth.GetServersMapOutput{Servers: m}, nil
}
