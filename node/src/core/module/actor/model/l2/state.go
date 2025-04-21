package l2

import "kasper/src/abstract/models"

type IStateL1 interface {
	Dummy() string
	Info() models.IInfo
	Trx() models.ITrx
	SetTrx(models.ITrx)
}

type State struct {
	info  models.IInfo
	trx   models.ITrx
	dummy string
}

func (s State) Info() models.IInfo {
	return s.info
}

func (s State) SetInfo(i models.IInfo) {
	s.info = i
}

func (s State) Trx() models.ITrx {
	return s.trx
}

func (s State) SetTrx(newTrx models.ITrx) {
	s.trx = newTrx
}

func (s State) Dummy() string {
	return s.dummy
}

func NewState(args ...interface{}) models.IState {
	var trx models.ITrx
	if (len(args) > 1) && (args[1] != nil) {
		trx = args[1].(models.ITrx)
	} else {
		trx = nil
	}

	if len(args) > 0 {
		if len(args) > 2 {
			return &State{info: args[0].(models.IInfo), trx: trx, dummy: args[2].(string)}
		} else {
			return &State{info: args[0].(models.IInfo), trx: trx, dummy: ""}
		}
	} else {
		return &State{info: nil, trx: trx, dummy: ""}
	}
}
