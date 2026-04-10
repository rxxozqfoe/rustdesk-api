package admin

type BuildJobTriggerForm struct {
	Version  string `json:"version" validate:"required"`
	Platform string `json:"platform" validate:"required"`
	Arch     string `json:"arch" validate:"required"`
	Format   string `json:"format" validate:"required"`
}

type BuildJobListQuery struct {
	Page     uint   `form:"page"`
	PageSize uint   `form:"page_size"`
	Status   string `form:"status"`
}
