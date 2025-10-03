package output_model_points

import model "applet/src/models"

type AdminPoiint struct {
	model.Point
	Admin bool `json:"admin"`
}

type CreateOutput struct {
	Point AdminPoiint `json:"point"`
}
