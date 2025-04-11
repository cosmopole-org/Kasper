package actions_auth

import (
	"kasper/src/abstract"
	inputsauth "kasper/src/shell/api/inputs/auth"
	outputsauth "kasper/src/shell/api/outputs/auth"
	"kasper/src/shell/layer1/adapters"
	tb "kasper/src/shell/layer1/module/toolbox"
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
		m = append(m, strings.Join(arr[0:len(arr)-1], ":"))
	}
	return outputsauth.GetServersMapOutput{Servers: m}, nil
}
