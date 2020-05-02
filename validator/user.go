package validator

// 所有返回必须包裹一层resp
type GlobalResp struct {
	Code    uint16      `json:"code"`    // 状态码
	Message string      `json:"message"` // 消息
	Data    interface{} `json:"data"`
}

// 用户登录
type UserLoginReq struct {
	UserName string `json:"user_name" comment:"用户名" validate:"required,max=20,min=3"`
	Password string `json:"password" comment:"密码" validate:"required,min=3,max=20"`
}

type UserLoginResp struct {
	Token string   `json:"token"`
	Roles []string `json:"roles"`
}
