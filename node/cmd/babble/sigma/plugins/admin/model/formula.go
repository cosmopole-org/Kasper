package admin_model

import "gorm.io/datatypes"

type MemFormula struct {
	Keys       []string                      `json:"keys"`
	Weights    []float64                     `json:"weights"`
	NonZero    []bool                        `json:"nonZero"`
	Operations []string                      `json:"operations"`
	Rules      map[string]map[string]float64 `json:"rules"`
	TotalOp    string                        `json:"totalOp"`
	Order      string                        `json:"order"`
}

type Formula struct {
	GameKey string     `json:"gameKey"`
	Level   string     `json:"level"`
	Data    MemFormula `json:"data"`
}

type StoredFormula struct {
	Id      string         `json:"id" gorm:"primaryKey;column:id"`
	GameKey string         `json:"gameKey" gorm:"column:game_key"`
	Level   string         `json:"level" gorm:"column:level"`
	Data    datatypes.JSON `json:"data" gorm:"column:data"`
}

func (m StoredFormula) Type() string {
	return "StoredFormula"
}