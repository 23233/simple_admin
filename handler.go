package simple_admin

import (
	"github.com/23233/simple_admin/v1/validator"
	"github.com/kataras/iris/v12"
	"strconv"
)

func Index(ctx iris.Context) {
	_ = ctx.View("index.html")
}

// 登录
func Login(ctx iris.Context) {
	var req validator.UserLoginReq
	// 引入数据
	if err := ctx.ReadJSON(&req); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}
	// 基础验证
	if err := validator.GlobalValidator.Check(req); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// 判断用户是否存在
	var valuesMap = make(map[string]string)
	has, err := NowSpAdmin.config.Engine.Table(NowSpAdmin.config.getUserModelTableName()).Where("user_name = ?", req.UserName).Get(&valuesMap)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": "get user requests fail",
		})
		return
	}
	if has == false {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": "not find user",
		})
		return
	}
	// 进行登录验证
	success := NowSpAdmin.config.validPassword(req.Password, valuesMap["salt"], valuesMap["password"])
	if success == false {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": "password fail",
		})
		return
	}
	// 判断用户是否有登录权限
	hasPolicy := NowSpAdmin.hasLoginPolicy(valuesMap["id"])
	if hasPolicy == false {
		ctx.StatusCode(iris.StatusUnauthorized)
		_, _ = ctx.JSON(iris.Map{
			"detail": "you account login prohibited",
		})
		return
	}
	// 生成jwt
	jwt := GenJwtToken(valuesMap["id"])
	var resp validator.UserLoginResp
	resp.Token = jwt
	_, _ = ctx.JSON(resp)
}

// 注册
func Reg(ctx iris.Context) {
	var req validator.UserLoginReq
	// 引入数据
	if err := ctx.ReadJSON(&req); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}
	// 基础验证
	if err := validator.GlobalValidator.Check(req); err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// 生成用户
	aff, err := NowSpAdmin.addUser(req.UserName, req.Password)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// 生成jwt
	jwt := GenJwtToken(strconv.FormatInt(aff, 10))
	var resp validator.UserLoginResp
	resp.Token = jwt
	_, _ = ctx.JSON(resp)
}
