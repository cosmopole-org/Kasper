package main

type Fn struct {
	UserId     string         `json:"userId"`
	Typ        string         `json:"type"`
	Username   string         `json:"username"`
	PublicKey  string         `json:"publicKey"`
	Name       string         `json:"name"`
	AppId      string         `json:"appId"`
	Runtime    string         `json:"runtime"`
	Path       string         `json:"path"`
	Comment    string         `json:"comment"`
	Identifier string         `json:"identifier"`
	Metadata   map[string]any `json:"metadata"`
}

type App struct {
	Id            string `json:"id"`
	ChainId       string `json:"chainId"`
	OwnerId       string `json:"ownerId"`
	Username      string `json:"username"`
	MachinesCount int    `json:"machinesCount"`
	Title         string `json:"title"`
	Avatar        string `json:"avatar"`
	Desc          string `json:"desc"`
}
