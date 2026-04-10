package api

/*
*

	message LoginRequest {
	  string username = 1;
	  bytes password = 2;
	  string my_id = 4;
	  string my_name = 5;
	  OptionMessage option = 6;
	  oneof union {
	    FileTransfer file_transfer = 7;
	    PortForward port_forward = 8;
	  }
	  bool video_ack_required = 9;
	  uint64 session_id = 10;
	  string version = 11;
	  OSLogin os_login = 12;
	  string my_platform = 13;
	  bytes hwid = 14;
	}
*/

type DeviceInfoInLogin struct {
	Name string `json:"name" label:"name"`
	Os   string `json:"os" label:"os"`
	Type string `json:"type" label:"type"`
}

type LoginForm struct {
	AutoLogin        bool              `json:"autoLogin" label:"自动登录"`
	DeviceInfo       DeviceInfoInLogin `json:"deviceInfo" label:"设备信息"`
	Id               string            `json:"id"  label:"id"`
	Type             string            `json:"type"  label:"type"`
	Uuid             string            `json:"uuid"  label:"uuid"`
	Username         string            `json:"username" validate:"required,gte=2,lte=32" label:"用户名"`
	Password         string            `json:"password,omitempty" validate:"gte=4,lte=32" label:"密码"`
	VerificationCode string            `json:"verificationCode,omitempty" label:"验证码"`
	TfaCode          string            `json:"tfaCode,omitempty" label:"2FA验证码"`
	Secret           string            `json:"secret,omitempty" label:"2FA secret"`
}

type DeviceIdentity struct {
	Id   string `json:"id"`
	Uuid string `json:"uuid"`
}

type UserListQuery struct {
	Page       uint   `json:"page" form:"page" label:"页码"`
	Current    uint   `json:"current" form:"current" label:"页码(别名)"`
	PageSize   uint   `json:"pageSize" form:"pageSize" validate:"required" label:"每页数量"`
	Status     int    `json:"status" form:"status" label:"状态"`
	Accessible string `json:"accessible" form:"accessible"`
}

func (q *UserListQuery) GetPage() uint {
	if q.Current > 0 {
		return q.Current
	}
	if q.Page > 0 {
		return q.Page
	}
	return 1
}

type PeerListQuery struct {
	Page       uint   `json:"page" form:"page" label:"页码"`
	Current    uint   `json:"current" form:"current" label:"页码(别名)"`
	PageSize   uint   `json:"pageSize" form:"pageSize" validate:"required" label:"每页数量"`
	Status     int    `json:"status" form:"status" label:"状态"`
	Accessible string `json:"accessible" form:"accessible"`
}

func (q *PeerListQuery) GetPage() uint {
	if q.Current > 0 {
		return q.Current
	}
	if q.Page > 0 {
		return q.Page
	}
	return 1
}
