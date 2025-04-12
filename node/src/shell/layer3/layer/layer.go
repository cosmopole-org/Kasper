package layer

import (
	"kasper/src/abstract"
	moduleactor "kasper/src/core/module/actor"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/layer1/module/toolbox"
	module_model "kasper/src/shell/layer2/model"
	modulemodel "kasper/src/shell/layer3/model"
	tool_net "kasper/src/shell/layer3/tools/network"
	netfederation "kasper/src/shell/layer3/tools/network/federation"
)

type Layer struct {
	core         abstract.ICore
	actor        abstract.IActor
	toolbox      abstract.IToolbox
	stateBuilder abstract.IStateBuilder
	logger       *modulelogger.Logger
	federation   *netfederation.FedNet
}

func New() abstract.ILayer {
	return &Layer{actor: moduleactor.NewActor()}
}

func (l *Layer) Core() abstract.ICore {
	return l.core
}

func (l *Layer) BackFill(core abstract.ICore, args ...interface{}) []interface{} {
	l.core = core
	l.logger = args[0].(*modulelogger.Logger)
	l.federation = netfederation.FirstStageBackFill(core, l.logger)

	return []interface{}{
		args[0], args[1], args[2], args[3], l.federation, args[4], args[5], args[6],
	}
}

func (l *Layer) ForFill(core abstract.ICore, args ...interface{}) {
	layer1Toolbox := abstract.UseToolbox[*toolbox.ToolboxL1](core.Get(1).Tools())
	net := tool_net.NewNetwork(core, l.logger, layer1Toolbox.Storage(), layer1Toolbox.Cache(), layer1Toolbox.Security(), layer1Toolbox.Signaler())
	net.Fed = l.federation.SecondStageForFill(net.Http, layer1Toolbox.Storage(), layer1Toolbox.Cache(), layer1Toolbox.Signaler())
	tb := modulemodel.NewTools(net)
	tb.ToolboxL2 = abstract.UseToolbox[*module_model.ToolboxL2](args[0])
	l.toolbox = tb
}

func (l *Layer) Index() int {
	return 0
}

func (l *Layer) Actor() abstract.IActor {
	return l.actor
}

func (l *Layer) Tools() abstract.IToolbox {
	return l.toolbox
}

func (l *Layer) Sb() abstract.IStateBuilder {
	return l.stateBuilder
}

func (l *Layer) InitSb(bottom abstract.IStateBuilder) abstract.IStateBuilder {
	l.stateBuilder = modulemodel.NewStateBuilder(l, bottom)
	return l.stateBuilder
}
