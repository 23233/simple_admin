package simple_admin

import (
	"fmt"
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

	// 获取用户拥有的角色 包含隐式继承
	roles, err := NowSpAdmin.casbinEnforcer.GetImplicitRolesForUser(valuesMap["id"])
	// 判断用户是否有登录权限
	hasPolicy := StringsContains(roles, NowSpAdmin.defaultRole["guest"])
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
	aff, err := NowSpAdmin.addUser(req.UserName, req.Password, NowSpAdmin.defaultRole["guest"])
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

// 获取所有表名
func GetRouters(ctx iris.Context) {
	uid := ctx.Values().Get("uid").(string)
	// 筛选当前用户可用的权限列表
	rules, err := NowSpAdmin.casbinEnforcer.GetImplicitPermissionsForUser(uid)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	result := make(map[string]string, 0)
	for _, rule := range rules {
		if rule[2] == NowSpAdmin.defaultMethods["GET"] && rule[1] != NowSpAdmin.sitePolicy["login_site"] {
			if rule[1] == NowSpAdmin.sitePolicy["user_manage"] {
				result["simple_admin_user_model"] = rule[1]
			} else {
				result[rule[1]] = rule[1]
			}
		}
	}
	_, _ = ctx.JSON(result)
}

// 获取单个表的列信息
func GetRouterFields(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	// todo:是否需要进行权限判断 待定?
	// 先确定有没有这个表
	if StringsContains(NowSpAdmin.modelTables, routerName) {
		fields, err := NowSpAdmin.config.tableNameToFieldAndTypes(routerName)
		if err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			_, _ = ctx.JSON(iris.Map{
				"detail": err.Error(),
			})
			return
		}
		_, _ = ctx.JSON(fields)
	} else {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": "params error ",
		})
		return
	}
}

// 获取表数据
func GetRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	page := ctx.URLParamIntDefault("page", 1)
	data, err := NowSpAdmin.Pagination(routerName, page)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(data)
}

// 获取表单条数据
func GetRouterSingleData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	id, err := ctx.Params().GetUint64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	data, err := NowSpAdmin.SingleData(routerName, id)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(data)
}

// 新增数据
func AddRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	newInstance, err := NowSpAdmin.getCtxValues(routerName, ctx)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	err = NowSpAdmin.addData(routerName, newInstance)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 修改数据 -> 全量更新
func EditRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	id, err := ctx.Params().GetUint64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// 获取ID 判断是否存在
	has, err := NowSpAdmin.dataExists(routerName, id)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	if has != true {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": fmt.Sprintf("not find this data %d", id),
		})
		return
	}
	newInstance, err := NowSpAdmin.getCtxValues(routerName, ctx)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// 更新数据
	err = NowSpAdmin.editData(routerName, id, newInstance)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 删除数据
func RemoveRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	id, err := ctx.Params().GetUint64("id")
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// 直接进行删除
	err = NowSpAdmin.deleteData(routerName, id)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 权限Middleware
func PolicyValidMiddleware(ctx iris.Context) {
	userUid := ctx.Values().Get("uid").(string)
	methods := ctx.Method()
	routerName := ctx.Params().Get("routerName")
	has, err := NowSpAdmin.casbinEnforcer.Enforce(userUid, routerName, methods)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	if has == false {
		ctx.StatusCode(iris.StatusForbidden)
		_, _ = ctx.JSON(iris.Map{
			"detail": "no permission to proceed",
		})
		return
	}
	ctx.Next()
}
