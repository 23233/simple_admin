package simple_admin

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"strconv"
)

func Index(ctx iris.Context) {
	rs := []rune(NowSpAdmin.config.Prefix)
	ctx.ViewData("prefix", string(rs[1:]))
	_ = ctx.View("simple_admin.template")
}

// 获取配置信息
func Configuration(ctx iris.Context) {
	var resp ConfigResp
	resp.Name = NowSpAdmin.config.Name
	resp.Prefix = NowSpAdmin.config.Prefix
	resp.UserModelName = NowSpAdmin.config.UserModelSpecialUniqueName
	_, _ = ctx.JSON(resp)
}

// 登录
func Login(ctx iris.Context) {
	req := ctx.Values().Get(SvKey).(*UserLoginReq)
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
	jwt := GenJwtToken(valuesMap["id"], req.UserName)
	var resp UserLoginResp
	resp.Token = jwt
	resp.UserName = req.UserName
	resp.Roles = roles
	_, _ = ctx.JSON(resp)
}

// 注册
func Reg(ctx iris.Context) {
	req := ctx.Values().Get(SvKey).(*UserLoginReq)

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
	jwt := GenJwtToken(strconv.FormatInt(aff, 10), req.UserName)
	var resp UserLoginResp
	resp.Token = jwt
	resp.UserName = req.UserName
	resp.Roles = []string{"guest"}
	_, _ = ctx.JSON(resp)
}

// 获取当前用户
func GetCurrentUser(ctx iris.Context) {
	un := ctx.Values().Get("un").(string)
	uid := ctx.Values().Get("uid").(string)
	var resp GetCurrentUserResp
	resp.Name = un
	resp.UserId = uid
	resp.Avatar = "https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png"
	_, _ = ctx.JSON(resp)
}

// 获取所有表名
func GetRouters(ctx iris.Context) {
	uid := ctx.Values().Get("uid").(string)
	result := make(map[string]string, 0)
	// 获取所有表名
	for _, table := range NowSpAdmin.modelTables {
		// 判断是否有读取权限
		has, err := NowSpAdmin.casbinEnforcer.Enforce(uid, table, NowSpAdmin.defaultMethods["GET"])
		if err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			_, _ = ctx.JSON(iris.Map{
				"detail": err.Error(),
			})
			return
		}
		if has == true {
			// 判断是否是用户模型
			if table == NowSpAdmin.sitePolicy["user_manage"] {
				result[NowSpAdmin.config.UserModelSpecialUniqueName] = table
			} else {
				result[table] = table
			}
		}
	}
	_, _ = ctx.JSON(result)
}

// 获取单个表的列信息
func GetRouterFields(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")

	fields, err := NowSpAdmin.config.tableNameReflectFieldsAndTypes(routerName)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(fields)
}

//获取单个表的自定义操作
func GetRouterCustomAction(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	action := NowSpAdmin.config.tableNameCustomActionScopeMatch(routerName)
	_, _ = ctx.JSON(action)
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

// 删除数据 -> 可以批量
func RemoveRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	req := ctx.Values().Get(SvKey).(*DeleteReq)
	// 进行批量删除
	err := NowSpAdmin.bulkDeleteData(routerName, req.Ids)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(iris.Map{})

}

// 变更用户密码
func ChangeUserPassword(ctx iris.Context) {
	req := ctx.Values().Get(SvKey).(*UserChangePasswordReq)
	uid := ctx.Values().Get("uid").(string)
	// 判断当前用户是否是admin权限
	roles, err := NowSpAdmin.casbinEnforcer.GetImplicitRolesForUser(uid)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	// admin可以变更所有 否则只能变更自己的密码
	if StringsContains(roles, NowSpAdmin.defaultRole["admin"]) == false {
		un := ctx.Values().Get("un").(string)
		if un != req.UserName {
			// 变更
			ctx.StatusCode(iris.StatusBadRequest)
			_, _ = ctx.JSON(iris.Map{
				"detail": "no permission to proceed",
			})
			return
		}
	}
	// 直接变更
	err = NowSpAdmin.changeUserPassword(req.Id, req.Password)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 变更用户群组
func ChangeUserRoles(ctx iris.Context) {
	req := ctx.Values().Get(SvKey).(*UserChangeRolesReq)
	var err error
	if req.Add {
		_, err = NowSpAdmin.casbinEnforcer.AddRoleForUser(strconv.FormatUint(req.Id, 10), req.Role)
	} else {
		_, err = NowSpAdmin.casbinEnforcer.DeleteRoleForUser(strconv.FormatUint(req.Id, 10), req.Role)
	}
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

//// todo: 变更用户权限
//func ChangeUserPolicy(ctx iris.Context) {
//
//}

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
		ctx.StatusCode(iris.StatusUnauthorized)
		_, _ = ctx.JSON(iris.Map{
			"detail": "no permission to proceed",
		})
		return
	}
	ctx.Next()
}

// 必须admin权限middleware
func PolicyRequireAdminMiddleware(ctx iris.Context) {
	uid := ctx.Values().Get("uid").(string)
	// 判断当前用户是否是admin权限
	roles, err := NowSpAdmin.casbinEnforcer.GetImplicitRolesForUser(uid)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(iris.Map{
			"detail": err.Error(),
		})
		return
	}
	if StringsContains(roles, NowSpAdmin.defaultRole["admin"]) == false {
		ctx.StatusCode(iris.StatusUnauthorized)
		_, _ = ctx.JSON(iris.Map{
			"detail": "no permission to proceed",
		})
		return
	}
	ctx.Next()
}
