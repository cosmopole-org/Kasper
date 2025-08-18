package model

type ListPointAppsOutput struct {
	Machines map[string]*Fn `json:"machines"`
	Apps     map[string]App `json:"apps"`
}
