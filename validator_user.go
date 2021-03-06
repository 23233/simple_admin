package simple_admin

// 用户登录
type UserLoginReq struct {
	UserName string `json:"user_name" comment:"用户名" validate:"required,max=20,min=3"`
	Password string `json:"password" comment:"密码" validate:"required,min=3,max=20"`
}

type UserLoginResp struct {
	Token    string             `json:"token"`
	UserInfo GetCurrentUserResp `json:"user_info"`
}

type GetCurrentUserResp struct {
	Name   string   `json:"name"`
	Avatar string   `json:"avatar"`
	UserId string   `json:"userid"`
	Roles  []string `json:"roles"`
}

// 用户变更密码
type UserChangePasswordReq struct {
	Id       uint64 `json:"id" comment:"id" validate:"required"`
	UserName string `json:"user_name" comment:"用户名" validate:"required,max=20,min=3"`
	Password string `json:"password" comment:"密码" validate:"required,min=3,max=20"`
}

// admin 变更用户群组
type UserChangeRolesReq struct {
	Id   uint64 `json:"id" comment:"id" validate:"required"`
	Role string `json:"role" comment:"群组名" validate:"required"`
	Add  bool   `json:"add" comment:"添加"`
}
