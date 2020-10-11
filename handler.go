package simple_admin

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/realip"
	"log"
	"strconv"
	"strings"
	"xorm.io/xorm"
)

// 错误返回
func fastError(err error, ctx iris.Context, msg ...string) {
	ctx.StatusCode(iris.StatusBadRequest)
	var m string
	if len(msg) >= 1 {
		m = msg[0]
	} else {
		if err == nil {
			m = "请求解析出错"
		} else {
			m = err.Error()
		}
	}
	_, _ = ctx.JSON(iris.Map{
		"detail": m,
	})
	return
}

// 首页
func Index(ctx iris.Context) {
	rs := []rune(NowSpAdmin.config.Prefix)
	ctx.ViewData("prefix", string(rs[1:]))
	_ = ctx.View("simple_admin")
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
		fastError(err, ctx, "获取用户请求错误")
		return
	}
	if has == false {
		fastError(err, ctx, "没有找到用户")
		return

	}
	// 进行登录验证
	success := NowSpAdmin.config.validPassword(req.Password, valuesMap["salt"], valuesMap["password"])
	if success == false {
		fastError(err, ctx, "密码错误")
		return
	}

	// 获取用户拥有的角色 包含隐式继承
	roles, err := NowSpAdmin.casbinEnforcer.GetImplicitRolesForUser(valuesMap["id"])
	// 判断用户是否有登录权限
	hasPolicy := StringsContains(roles, NowSpAdmin.defaultRole["guest"])
	if hasPolicy == false {
		ctx.StatusCode(iris.StatusUnauthorized)
		_, _ = ctx.JSON(iris.Map{
			"detail": "你的账户已被禁止登录,请联系管理员",
		})
		return
	}
	// 生成jwt
	jwt := GenJwtToken(valuesMap["id"], req.UserName)
	var resp UserLoginResp
	resp.Token = jwt
	_, _ = ctx.JSON(resp)
}

// 注册
func Reg(ctx iris.Context) {
	req := ctx.Values().Get(SvKey).(*UserLoginReq)
	// 生成用户
	aff, err := NowSpAdmin.addUser(req.UserName, req.Password, NowSpAdmin.defaultRole["guest"])
	if err != nil {
		fastError(err, ctx)
		return
	}
	// 生成jwt
	jwt := GenJwtToken(strconv.FormatInt(aff, 10), req.UserName)
	var resp UserLoginResp
	resp.Token = jwt
	_, _ = ctx.JSON(resp)
}

// 获取当前用户
func GetCurrentUser(ctx iris.Context) {
	un := ctx.Values().Get("un").(string)
	uid := ctx.Values().Get("uid").(string)
	roles, _ := NowSpAdmin.casbinEnforcer.GetImplicitRolesForUser(uid)
	var resp GetCurrentUserResp
	resp.Name = un
	resp.UserId = uid
	resp.Avatar = "https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png"
	resp.Roles = roles
	_, _ = ctx.JSON(resp)
}

// 获取所有表名
func GetRouters(ctx iris.Context) {
	uid := ctx.Values().Get("uid").(string)

	var result GetAllTableNameResp

	var names = make(map[string]string, 0)
	var remarks = make(map[string]string, 0)
	for _, m := range NowSpAdmin.config.modelInfoList {
		// 判断是否有读取权限
		has, err := NowSpAdmin.casbinEnforcer.Enforce(uid, m.RouterName, NowSpAdmin.defaultMethods["GET"])
		if err != nil {
			fastError(err, ctx)
			return
		}
		if has == true {
			// 判断是否是用户模型
			if m.RouterName == NowSpAdmin.sitePolicy["user_manage"] {
				names[NowSpAdmin.config.UserModelSpecialUniqueName] = m.RouterName
				remarks[NowSpAdmin.config.UserModelSpecialUniqueName] = "后台用户表"
			} else {
				names[m.RouterName] = m.RouterName
				if len(m.RemarkName) >= 1 {
					remarks[m.RouterName] = m.RemarkName
				} else {
					remarks[m.RouterName] = m.RouterName
				}
			}
		}
	}
	result.Tables = names
	result.Remarks = remarks
	_, _ = ctx.JSON(result)
}

// 获取单个表的列信息
func GetRouterFields(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")

	cb, err := NowSpAdmin.config.tableNameGetModelInfo(routerName)
	if err != nil {
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(cb.FieldList)
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
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(data)
}

// 搜索表数据
func SearchRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	req := ctx.Values().Get(SvKey).(*SearchReq)
	data, err := NowSpAdmin.searchData(routerName, req.SearchText, req.Cols, req.FullMath)
	if err != nil {
		fastError(err, ctx)
		return
	}

	_, _ = ctx.JSON(data)
}

// 获取表单条数据
func GetRouterSingleData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	id, err := ctx.Params().GetUint64("id")
	if err != nil {
		fastError(err, ctx)
		return
	}
	data, err := NowSpAdmin.SingleData(routerName, id)
	if err != nil {
		fastError(err, ctx)
		return
	}
	if len(data) >= 1 {
		_, _ = ctx.JSON(data[0])
	} else {
		_, _ = ctx.JSON(data)
	}
}

// 新增数据
func AddRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	newInstance, err := NowSpAdmin.getCtxValues(routerName, ctx)
	if err != nil {
		fastError(err, ctx)
		return
	}
	err = NowSpAdmin.addData(routerName, newInstance)
	if err != nil {
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 修改数据 -> 全量更新
func EditRouterData(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	id, err := ctx.Params().GetUint64("id")
	if err != nil {
		fastError(err, ctx)
		return
	}
	// 获取ID 判断是否存在
	has, err := NowSpAdmin.dataExists(routerName, id)
	if err != nil {
		fastError(err, ctx)
		return
	}
	if has != true {
		fastError(err, ctx, fmt.Sprintf("not find this data %d", id))
		return
	}
	newInstance, err := NowSpAdmin.getCtxValues(routerName, ctx)
	if err != nil {
		fastError(err, ctx)
		return
	}
	// 更新数据
	err = NowSpAdmin.editData(routerName, id, newInstance)
	if err != nil {
		fastError(err, ctx)
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
		fastError(err, ctx)
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
		fastError(err, ctx)
		return
	}
	// admin可以变更所有 否则只能变更自己的密码
	if StringsContains(roles, NowSpAdmin.defaultRole["admin"]) == false {
		un := ctx.Values().Get("un").(string)
		if un != req.UserName {
			fastError(err, ctx, "无权限进行操作")
			return
		}
	}
	// 直接变更
	err = NowSpAdmin.changeUserPassword(req.Id, req.Password)
	if err != nil {
		fastError(err, ctx)
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
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

//// todo: 变更用户权限
//func ChangeUserPolicy(ctx iris.Context) {
//
//}

// 数据可视化屏幕
func GetDashBoardScreen(ctx iris.Context) {
	userId := ctx.Values().Get("uid").(string)
	result := make([]DashBoardScreen, 0)
	err := NowSpAdmin.config.Engine.Where("create_user_id = ?", userId).Find(&result)
	if err != nil {
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(result)
}

func AddDashBoardScreen(ctx iris.Context) {
	userId := ctx.Values().Get("uid").(string)
	req := ctx.Values().Get(SvKey).(*DashBoardScreenAddOrEditReq)
	var single DashBoardScreen
	if req.Id >= 1 {
		// 修改
		has, err := NowSpAdmin.config.Engine.Where("create_user_id = ? and id = ?", userId, req.Id).Get(&single)
		if err != nil {
			fastError(err, ctx)
			return
		}
		if has == false {
			fastError(err, ctx, "数据获取失败")
			return
		}
		single.Name = req.Name
		single.IsDefault = req.IsDefault
		aff, err := NowSpAdmin.config.Engine.ID(single.Id).Cols("name", "is_default").Update(&single)
		if err != nil || aff == 0 {
			fastError(err, ctx, "更新数据失败")
			return
		}
	} else {
		single.Name = req.Name
		single.IsDefault = req.IsDefault
		id, _ := strconv.Atoi(userId)
		single.CreateUserId = uint64(id)
		aff, err := NowSpAdmin.config.Engine.InsertOne(&single)
		if err != nil || aff == 0 {
			fastError(err, ctx, "新增数据失败")
			return
		}
	}
	_, _ = ctx.JSON(iris.Map{})

}

func DeleteBoardScreen(ctx iris.Context) {
	userId := ctx.Values().Get("uid").(string)
	id, err := ctx.Params().GetUint64("id")
	if err != nil {
		fastError(err, ctx)
		return
	}
	var d DashBoardScreen
	aff, err := NowSpAdmin.config.Engine.Where("create_user_id = ? and id = ?", userId, id).Delete(&d)
	if err != nil || aff < 1 {
		fastError(err, ctx, "删除数据失败")
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 数据图表
func GetDashBoard(ctx iris.Context) {
	screenId, _ := ctx.Params().GetUint64("id")
	result := make([]DashBoard, 0)
	err := NowSpAdmin.config.Engine.Where("screen_id = ?", screenId).Find(&result)
	if err != nil {
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(result)

}

// 新增图表
func AddDashBoard(ctx iris.Context) {
	req := ctx.Values().Get(SvKey).(*DashBoardAddReq)
	screenId, _ := ctx.Params().GetUint64("id")
	var single DashBoard
	single.ScreenId = screenId
	single.Name = req.Name
	single.ChartType = req.ChartType
	single.DataSource = req.DataSource
	single.Config = req.Config
	aff, err := NowSpAdmin.config.Engine.InsertOne(&single)
	if err != nil || aff == 0 {
		fastError(err, ctx, "新增数据失败")
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 删除图表
func DeleteDashBoard(ctx iris.Context) {
	id, err := ctx.Params().GetUint64("rid")
	if err != nil {
		fastError(err, ctx)
		return
	}
	var d DashBoard
	aff, err := NowSpAdmin.config.Engine.ID(id).Delete(&d)
	if err != nil || aff < 1 {
		fastError(err, ctx, "删除数据失败")
		return
	}
	_, _ = ctx.JSON(iris.Map{})
}

// 获取图表数据源
func DashBoardSourceGet(ctx iris.Context) {
	routerName := ctx.Params().Get("routerName")
	userUid := ctx.Values().Get("uid").(string)
	has, err := NowSpAdmin.casbinEnforcer.Enforce(userUid, routerName, "GET")
	if err != nil {
		fastError(err, ctx)
		return
	}
	if has == false {
		ctx.StatusCode(iris.StatusMethodNotAllowed)
		_, _ = ctx.JSON(iris.Map{
			"detail": "无权操作,请联系管理员",
		})
		return
	}
	req := ctx.Values().Get(SvKey).(*DashBoardGetDataReq)
	base := func() *xorm.Session {
		return NowSpAdmin.config.Engine.Table(routerName)
	}
	var run = base()
	if len(req.ColumnOp) >= 1 {
		for i, item := range req.ColumnOp {
			sql := fmt.Sprintf("%s %s ?", item.ColName, item.OpType)
			if StringsContains([]string{"in", "not in"}, item.OpType) {
				if item.OpType == "in" {
					run = run.In(item.ColName, strings.Split(item.Value, ","))
				} else {
					run = run.NotIn(item.ColName, strings.Split(item.Value, ","))
				}
				continue
			}
			if len(item.ConnectType) >= 1 {
				if i == 0 {
					run = run.Where(sql, item.Value)
					continue
				}
				switch item.ConnectType {
				case "and":
					run = run.And(sql, item.Value)
					break
				case "or":
					run = run.Or(sql, item.Value)
					break
				default:
					break
				}
			} else {
				run = run.Where(sql, item.Value)
			}
		}
	}
	if req.Limit > 0 {
		run = run.Limit(int(req.Limit))
	}
	dataList, err := run.QueryString()
	if err != nil {
		fastError(err, ctx)
		return
	}
	_, _ = ctx.JSON(dataList)
}

// 爬虫监听Middleware
func SpiderVisitHistoryMiddleware(ctx iris.Context) {
	// 如果开启了监听
	if NowSpAdmin.config.EnableSpiderWatch {
		go func() {
			ua := strings.ToLower(ctx.GetHeader("User-Agent"))
			// 判断是否跳过
			for _, prefix := range NowSpAdmin.config.SpiderSkipList {
				if strings.Contains(ua, prefix) {
					return
				}
			}
			// 判断ua是否是爬虫
			for _, prefix := range NowSpAdmin.config.SpiderMatchList {
				if strings.Contains(ua, prefix) {
					ip := realip.Get(ctx.Request())
					var d SpiderHistory
					d.Ip = ip
					d.Ua = ua
					d.Match = prefix
					d.Page = ctx.Path()
					aff, err := NowSpAdmin.config.Engine.InsertOne(&d)
					if err != nil || aff != 1 {
						log.Printf("add spider visit history fail %v ", err)
					}
					return
				}
			}
		}()
	}
	ctx.Next()
}

// 权限Middleware
func PolicyValidMiddleware(ctx iris.Context) {
	userUid := ctx.Values().Get("uid").(string)
	methods := ctx.Method()
	routerName := ctx.Params().Get("routerName")
	has, err := NowSpAdmin.casbinEnforcer.Enforce(userUid, routerName, methods)
	if err != nil {
		fastError(err, ctx)
		return
	}
	if has == false {
		ctx.StatusCode(iris.StatusMethodNotAllowed)
		_, _ = ctx.JSON(iris.Map{
			"detail": "无权操作,请联系管理员",
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
		fastError(err, ctx)
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

// 数据可视化必须本人操作Middleware
func DashBoardIsSelfMiddleware(ctx iris.Context) {
	userId := ctx.Values().Get("uid").(string)
	screenId, err := ctx.Params().GetUint64("id")
	if err != nil {
		fastError(err, ctx)
		return
	}
	// 判断用户是否具有数据屏幕权限
	has, err := NowSpAdmin.config.Engine.Where("id = ? and create_user_id = ?", screenId, userId).Exist(new(DashBoardScreen))
	if err != nil || has == false {
		fastError(err, ctx, "获取图表数据失败")
		return
	}
	ctx.Next()
}
