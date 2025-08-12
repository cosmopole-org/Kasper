package inputs_storage

type DownloadUserEntityInput struct {
	EntityId string `json:"entityId" validate:"required"`
}

func (d DownloadUserEntityInput) GetData() any {
	return "dummy"
}

func (d DownloadUserEntityInput) GetPointId() string {
	return ""
}

func (d DownloadUserEntityInput) Origin() string {
	return ""
}
