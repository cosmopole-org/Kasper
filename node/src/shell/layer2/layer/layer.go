package layer

import (
	"kasper/src/abstract"
	moduleactor "kasper/src/core/module/actor"
	modulelogger "kasper/src/core/module/logger"
	toolbox2 "kasper/src/shell/layer1/module/toolbox"
	modulemodel "kasper/src/shell/layer2/model"
	toolcache "kasper/src/shell/layer2/tools/cache"
	toolfile "kasper/src/shell/layer2/tools/file"
	toolstorage "kasper/src/shell/layer2/tools/storage"

	"gorm.io/gorm"
)

type Layer struct {
	core         abstract.ICore
	actor        abstract.IActor
	toolbox      abstract.IToolbox
	stateBuilder abstract.IStateBuilder
}

func New() abstract.ILayer {
	return &Layer{actor: moduleactor.NewActor()}
}

func (l *Layer) Core() abstract.ICore {
	return l.core
}

func (l *Layer) BackFill(core abstract.ICore, args ...interface{}) []interface{} {
	l.core = core
	cache := toolcache.NewCache(core, args[0].(*modulelogger.Logger), args[3].(string))
	storage := toolstorage.NewStorage(core, cache, args[0].(*modulelogger.Logger), args[1].(string), args[2].(gorm.Dialector))
	file := toolfile.NewFileTool(args[0].(*modulelogger.Logger))
	l.toolbox = modulemodel.NewTools(core, args[0].(*modulelogger.Logger), args[1].(string), storage, args[7].(string), cache, file)
	return []interface{}{
		args[0],
		storage,
		cache,
		args[4],
		args[5],
		args[6],
	}
}

func (l *Layer) ForFill(_ abstract.ICore, args ...interface{}) {
	toolbox := abstract.UseToolbox[*modulemodel.ToolboxL2](l.toolbox)
	toolbox.ToolboxL1 = abstract.UseToolbox[*toolbox2.ToolboxL1](args[0])
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
