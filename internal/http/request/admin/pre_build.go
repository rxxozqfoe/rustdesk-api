package admin

type PreBuildTriggerForm struct {
	Version  string `json:"version" validate:"required"`
	Platform string `json:"platform" validate:"required"`
	Arch     string `json:"arch" validate:"required"`
}

type PreBuildListQuery struct {
	Page     uint   `form:"page"`
	PageSize uint   `form:"page_size"`
	Status   string `form:"status"`
}
