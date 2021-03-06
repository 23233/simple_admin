package simple_admin

import (
	"fmt"
	"github.com/23233/sv"
	"github.com/casbin/casbin/v2"
	"github.com/imdario/mergo"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/router"
	"github.com/pkg/errors"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
	"xorm.io/xorm"
)

var (
	NowSpAdmin    *SpAdmin
	SvKey         = sv.GlobalContextKey
	DefaultPrefix = "/SP_PREFIX"
)

type SpAdmin struct {
	config         Config
	casbinEnforcer *casbin.Enforcer
	modelTables    []string
	defaultMethods map[string]string // 默认权限方法
	defaultRole    map[string]string // 默认角色
	sitePolicy     map[string]string
	adminPolicy    []string
	prefix         string
}

func (lib *SpAdmin) errorLog(err error, msg string) error {
	return errors.Wrap(err, msg)
}

func New(c Config) (*SpAdmin, error) {
	// 合并配置文件
	newConf := new(Config).initConfig()
	if err := mergo.Map(&c, newConf); err != nil {
		return nil, err
	}
	// 进行初始化处理
	if err := c.valid(); err != nil {
		return nil, err
	}
	// 初始化
	c.scanTableInfo()
	// 对表进行一次基础信息捕获
	c.generateTables()

	// 进行初始化权限系统
	enforcer, err := c.initCasBin()
	if err != nil {
		return nil, err
	}

	// 进行sync操作
	if err := c.runSync(); err != nil {
		return nil, err
	}

	// 生成表名列表
	modelTables := c.generateTables()

	NowSpAdmin = &SpAdmin{
		config:         c,
		casbinEnforcer: enforcer,
		modelTables:    modelTables,
		defaultMethods: map[string]string{
			"GET":    "GET",
			"POST":   "POST",
			"PUT":    "PUT",
			"DELETE": "DELETE",
		},
		defaultRole: map[string]string{
			"guest": "guest",
			"staff": "staff",
			"admin": "admin",
		},
		sitePolicy: map[string]string{
			"login_site":  "login_site",
			"user_manage": c.getUserModelTableName(),
		},
		adminPolicy: []string{c.Engine.TableName(new(DashBoardScreen)), c.Engine.TableName(new(DashBoard)), c.getUserModelTableName()},
		prefix:      DefaultPrefix,
	}
	// 进行视图注册绑定
	NowSpAdmin.Register()

	// 初始化权限
	err = NowSpAdmin.initRolesAndPermissions()
	if err != nil {
		return nil, err
	}

	// 初始化管理员
	_, err = NowSpAdmin.addUser(c.InitAdminUserName, c.InitAdminPassword, NowSpAdmin.defaultRole["admin"])
	if err != nil {
		log.Printf("initConfig admin user fail: %s", err.Error())
	}

	return NowSpAdmin, nil
}

// 在这里注册主路由
func (lib *SpAdmin) Router(router iris.Party) {
	router.RegisterView(iris.Blocks(AssetFile(), ".html"))
	// 首页
	router.Get("/", Index)
	// 登录
	router.Post("/login", sv.Run(new(UserLoginReq)), Login)
	// 注册
	router.Post("/reg", sv.Run(new(UserLoginReq)), Reg)
	router.Get("/config", Configuration)

	b := router.Party("/b", CustomJwt.Serve, TokenToUserUidMiddleware)
	// 数据可视化屏幕
	b.Get("/dash_board_screen", GetDashBoardScreen)
	b.Post("/dash_board_screen", sv.Run(new(DashBoardScreenAddOrEditReq)), AddDashBoardScreen)
	b.Delete("/dash_board_screen/{id:uint64}", DeleteBoardScreen)
	// 数据可视化 图表
	b.Get("/data_board/{id:uint64}", DashBoardIsSelfMiddleware, GetDashBoard)
	b.Get("/data_board/{id:uint64}/{rid:uint64}", DashBoardIsSelfMiddleware, GetSingleDashBoard)
	b.Post("/data_board/{id:uint64}", sv.Run(new(DashBoardAddReq)), DashBoardIsSelfMiddleware, AddDashBoard)
	b.Put("/data_board/{id:uint64}/{rid:uint64}", sv.Run(new(DashBoardAddReq)), DashBoardIsSelfMiddleware, EditDashBoard)
	b.Put("/data_board_size/{id:uint64}/{rid:uint64}", sv.Run(new(DashBoardChangePositionReq)), DashBoardIsSelfMiddleware, EditDashBoardPosition)
	b.Delete("/data_board/{id:uint64}/{rid:uint64}", DashBoardIsSelfMiddleware, DeleteDashBoard)
	b.Post("/data_board_data/{routerName:string}", sv.Run(new(DashBoardGetDataReq)), DashBoardSourceGet)

	c := router.Party("/v", CustomJwt.Serve, TokenToUserUidMiddleware)
	// 获取当前用户
	c.Get("/get_current_user", GetCurrentUser)
	// 变更用户密码
	c.Post("/change_password", sv.Run(new(UserChangePasswordReq)), ChangeUserPassword)
	// 获取所有表
	c.Get("/get_routers", GetRouters)
	// 获取单表列信息
	c.Get("/get_routers_fields/{routerName:string}", PolicyValidMiddleware, GetRouterFields)
	// 获取单表自定义action
	c.Get("/get_routers_action/{routerName:string}", PolicyValidMiddleware, GetRouterCustomAction)
	// 查看
	c.Get("/{routerName:string}", PolicyValidMiddleware, GetRouterData)
	c.Post("/{routerName:string}/search", sv.Run(new(SearchReq)), SearchRouterData)
	c.Get("/{routerName:string}/{id:uint64}", PolicyValidMiddleware, GetRouterSingleData)
	// 增加
	c.Post("/{routerName:string}", PolicyValidMiddleware, AddRouterData)
	// 修改
	c.Put("/{routerName:string}/{id:uint64}", PolicyValidMiddleware, EditRouterData)
	// 删除 delete模式在某些匹配时候有问题
	c.Post("/{routerName:string}/delete", PolicyValidMiddleware, sv.Run(new(DeleteReq)), RemoveRouterData)
	// 权限相关
	c.Post("/change_user_role", PolicyRequireAdminMiddleware, sv.Run(new(UserChangeRolesReq)), ChangeUserRoles)
	// 进行自定义action绑定
	for _, m := range lib.config.modelInfoList {
		if len(m.Actions) >= 1 {
			for _, action := range m.Actions {
				if action.hasValid == false {
					c.Handle(action.Methods, action.Path, action.Func)
				} else {
					c.Handle(action.Methods, action.Path, sv.Run(action.Valid), action.Func)
				}
			}
		}
	}
}

// 权限变更
func (lib *SpAdmin) policyChange(userId, path, methods string, add bool) error {
	if add {
		// 先判断权限是否存在
		if lib.casbinEnforcer.HasPolicy(userId, path, methods) {
			return MsgLog("policy has exists")
		}
		success, err := lib.casbinEnforcer.AddPolicy(userId, path, methods)
		if err != nil || success == false {
			return MsgLog(fmt.Sprintf("add policy fail -> %s %s %s err:%s", userId, path, methods, err))
		}
		return nil
	}
	success, err := lib.casbinEnforcer.RemovePolicy(userId, path, methods)
	if err != nil || success == false {
		return MsgLog(fmt.Sprintf("remove policy fail -> %s %s %s err:%s", userId, path, methods, err))
	}
	return nil

}

// 获取权限 根据注册model filterMethods only needs methods data
func (lib *SpAdmin) getAllPolicy(userIdOrRoleName string, filterMethods []string, excludeModelTable ...string) [][]string {
	policyList := make([][]string, 0, (len(lib.modelTables)+len(lib.sitePolicy))*len(lib.defaultMethods))
	var d []string
	for _, v := range lib.sitePolicy {
		d = append(d, v)
	}
	full := append(lib.modelTables, d...)
	for _, item := range full {
		if len(item) >= 1 {
			if StringsContains(excludeModelTable, item) {
				continue
			}
			for _, method := range lib.defaultMethods {
				if StringsContains(filterMethods, method) {
					policyList = append(policyList, []string{userIdOrRoleName, item, method})
				}
			}
		}
	}
	return policyList
}

// 新建用户
func (lib *SpAdmin) addUser(userName, password string, role string) (int64, error) {
	values := GetMapValues(lib.defaultRole)
	if StringsContains(values, role) == false {
		return 0, MsgLog(fmt.Sprintf("role params not in %s", values))
	}
	ps, salt := lib.config.passwordSalt(password)
	// 获取表名
	tableName := lib.config.getUserModelTableName()
	// 判断用户是否存在
	has, err := lib.config.Engine.Table(tableName).Where("user_name = ?", userName).Exist()
	if has == true {
		return 0, MsgLog("user has exist")
	}
	if err != nil {
		return 0, err
	}
	// 新增用户
	success, err := lib.config.Engine.Exec(fmt.Sprintf("insert into %s (user_name,password,salt) VALUES (?,?,?)", tableName), userName, ps, salt)
	if err != nil {
		return 0, err
	}
	userUid, err := success.LastInsertId()
	if userUid == 0 || err != nil {
		return 0, MsgLog("insert user fail")
	}

	uid := strconv.FormatInt(userUid, 10)

	// 把用户写入群组
	stats, err := lib.casbinEnforcer.AddRoleForUser(uid, role)
	if stats != true || err != nil {
		return 0, MsgLog(fmt.Sprintf("add user to role:%s fail %s", role, err))
	}
	return userUid, nil
}

// 初始化权限和角色 颗粒度粗放
func (lib *SpAdmin) initRolesAndPermissions() error {
	// 先创建角色
	for _, role := range lib.defaultRole {
		switch role {
		case "guest":
			// 来宾只能登录
			_, err := lib.casbinEnforcer.AddPermissionForUser(role, lib.sitePolicy["login_site"], "POST")
			if err != nil {
				return MsgLog(fmt.Sprintf("initConfig guest role fail %s", err))
			}
			break
		case "staff":
			// 职员只能看
			rules := lib.getAllPolicy(role, []string{"GET"}, lib.adminPolicy...)
			for _, rule := range rules {
				if rule != nil {
					_, err := lib.casbinEnforcer.AddPermissionForUser(role, rule[1], rule[2])
					if err != nil {
						return MsgLog(fmt.Sprintf("initConfig staff role fail %s", err))
					}
				}
			}
			break
		case "admin":
			// 所有都能干
			rules := lib.getAllPolicy("admin", []string{"POST", "PUT", "DELETE"})
			for _, rule := range rules {
				if rule != nil {
					_, err := lib.casbinEnforcer.AddPermissionForUser(role, rule[1], rule[2])
					if err != nil {
						return MsgLog(fmt.Sprintf("initConfig admin role fail %s", err))
					}
				}
			}
			// 专属的管理员控制
			for _, value := range lib.defaultMethods {
				for _, s := range lib.adminPolicy {
					_, err := lib.casbinEnforcer.AddPermissionForUser(role, s, value)
					if err != nil {
						return MsgLog(fmt.Sprintf("initConfig admin user manage fail  %s", err))
					}
				}
			}
			break
		}
	}
	// 创建角色继承
	_, err := lib.casbinEnforcer.AddRoleForUser(lib.defaultRole["admin"], lib.defaultRole["staff"])
	if err != nil {
		return MsgLog(fmt.Sprintf("role admin has stfall fail  %s", err))
	}
	_, err = lib.casbinEnforcer.AddRoleForUser(lib.defaultRole["staff"], lib.defaultRole["guest"])
	if err != nil {
		return MsgLog(fmt.Sprintf("role staff has guest fail %s", err))
	}
	return nil
}

// 分页
func (lib *SpAdmin) Pagination(routerName string, page int) (PagResult, error) {
	var p PagResult
	start := (page - 1) * lib.config.PageSize
	end := page * (lib.config.PageSize * 2)
	// 先获取总数量
	allCount, err := lib.config.Engine.Table(routerName).Count()
	if err != nil {
		return p, err
	}

	dataList, err := lib.config.Engine.Table(routerName).And("id between ? and ?", start, end).Limit(lib.config.PageSize).QueryString()
	if err != nil {
		return p, err
	}

	// 如果是用户表 还需要返回当前的权限
	// 只有admin 才能请求到用户表
	cb, err := lib.config.tableNameGetModelInfo(routerName)
	if err != nil {
		return p, err
	}
	if routerName == lib.config.getUserModelTableName() {
		for i, d := range dataList {
			for k, v := range d {
				if k == cb.FieldList.AutoIncrement {
					roles, err := lib.casbinEnforcer.GetImplicitRolesForUser(v)
					if err != nil {
						return p, err
					}
					dataList[i]["roles"] = strings.Join(roles, ",")
					break
				}
			}
		}
	}

	p.PageSize = lib.config.PageSize
	p.Page = page
	p.All = allCount
	p.Data = dataList
	return p, nil

}

// 单条数据获取
func (lib *SpAdmin) SingleData(routerName string, id uint64) ([]map[string]string, error) {
	valuesMap, err := lib.config.Engine.Table(routerName).Where("id = ?", id).QueryString()
	if err != nil {
		return valuesMap, err
	}
	return valuesMap, nil
}

// 新增数据
func (lib *SpAdmin) addData(routerName string, data reflect.Value) error {
	singleData := data.Interface()

	// 插入之前的事件
	if processor, ok := singleData.(SpInsertBeforeProcess); ok {
		processor.SpInsertBefore()
	}

	aff, err := lib.config.Engine.Table(routerName).InsertOne(singleData)
	if aff == 0 || err != nil {
		return MsgLog(fmt.Sprintf("insert data fail %s", err))
	}

	// 插入之后的事件
	if processor, ok := singleData.(SpInsertAfterProcess); ok {
		processor.SpInsertAfter()
	}

	// 获取
	return nil
}

// 数据修改
func (lib *SpAdmin) editData(routerName string, id uint64, data reflect.Value) error {
	singleData := lib.incrSetValue(data, routerName, id).Interface()

	// 更新之前的事件
	if processor, ok := singleData.(SpUpdateBeforeProcess); ok {
		processor.SpUpdateBefore()
	}

	// 默认只更新非空和非0的字段 xorm的规则
	// 所以这里启动全量更新 传入数据必须为全量
	aff, err := lib.config.Engine.Table(routerName).ID(id).AllCols().Update(singleData)
	if aff == 0 || err != nil {
		return MsgLog(fmt.Sprintf("edit data fail %s id:%d router:%s", err, id, routerName))
	}

	// 更新之后的事件
	if processor, ok := singleData.(SpUpdateAfterProcess); ok {
		processor.SpUpdateAfter()
	}

	// 获取
	return nil
}

// 数据删除
func (lib *SpAdmin) deleteData(routerName string, id uint64) error {
	model, _ := lib.config.tableNameGetModel(routerName)
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	newInstance := reflect.New(t).Interface()
	// 找到这条数据
	has, err := lib.config.Engine.Table(newInstance).ID(id).Get(newInstance)
	if err != nil {
		return err
	}
	if has == false {
		return errors.New("未找到此数据")
	}
	// 删除之前的事件

	if processor, ok := newInstance.(SpDeleteBeforeProcess); ok {
		processor.SpDeleteBefore()
	}

	aff, err := lib.config.Engine.ID(id).Unscoped().Delete(model)
	if err != nil {
		return err
	}
	if aff < 1 {
		return MsgLog("删除数据失败")
	}

	if processor, ok := newInstance.(SpDeleteAfterProcess); ok {
		processor.SpDeleteAfter()
	}

	return nil
}

// 批量数据删除
func (lib *SpAdmin) bulkDeleteData(routerName string, ids string) error {
	idList := strings.Split(ids, ",")
	for _, item := range idList {
		id, err := strconv.Atoi(item)
		if err != nil {
			return err
		}
		err = lib.deleteData(routerName, uint64(id))
		if err != nil {
			return err
		}
	}

	return nil
}

// 搜索数据
func (lib *SpAdmin) searchData(routerName string, searchText string, columnMapName []string, fullMath bool) ([]map[string]string, error) {
	var result = make([]map[string]string, 0)

	whereJoin := make([]string, 0)
	for _, field := range columnMapName {
		whereJoin = append(whereJoin, fmt.Sprintf("`%s` like ?", field))
	}
	base := func() *xorm.Session {
		return lib.config.Engine.Table(routerName).Limit(20)
	}
	var run = base()
	for _, s := range whereJoin {
		if fullMath {
			run = run.Or(s, "%"+searchText+"%")
		} else {
			run = run.Or(s, searchText+"%")
		}
	}

	result, err := run.QueryString()
	if err != nil {
		return result, err
	}
	return result, nil
}

// 判断数据是否存在
func (lib *SpAdmin) dataExists(routerName string, id uint64) (bool, error) {
	return lib.config.Engine.Table(routerName).Where("id = ?", id).Exist()
}

// 获取内容
func (lib *SpAdmin) getValue(ctx iris.Context, k string) string {
	c := ctx.PostValueTrim(k)
	if len(c) < 1 {
		c = ctx.FormValue(k)
	}
	return c
}

// 对应关系获取
func (lib *SpAdmin) getCtxValues(routerName string, ctx iris.Context) (reflect.Value, error) {
	// 先获取到字段信息
	model, err := lib.config.tableNameGetModel(routerName)
	cb, err := lib.config.tableNameGetModelInfo(routerName)
	if err != nil {
		return reflect.Value{}, err
	}
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	newInstance := reflect.New(t)

	for _, column := range cb.FieldList.Fields {
		if column.MapName != cb.FieldList.AutoIncrement {
			if column.MapName == cb.FieldList.Updated || column.MapName == cb.FieldList.Deleted {
				continue
			}
			if len(cb.FieldList.Created) >= 1 {
				var equal = false
				for k := range cb.FieldList.Created {
					if column.MapName == k {
						equal = true
						break
					}
				}
				if equal {
					continue
				}
			}
			content := NowSpAdmin.getValue(ctx, column.MapName)
			switch column.Types {
			case "string":
				newInstance.Elem().FieldByName(column.Name).SetString(content)
				continue
			case "int", "int8", "int16", "int32", "int64", "time.Duration":
				d, err := strconv.ParseInt(content, 10, 64)
				if err != nil {
					log.Printf("解析出int出错")
				}
				newInstance.Elem().FieldByName(column.Name).SetInt(d)
				continue
			case "uint", "uint8", "uint16", "uint32", "uint64":
				d, err := strconv.ParseUint(content, 10, 64)
				if err != nil {
					log.Println("解析出uint出错")
				}
				newInstance.Elem().FieldByName(column.Name).SetUint(d)
				continue
			case "float32", "float64":
				d, err := strconv.ParseFloat(content, 64)
				if err != nil {
					log.Println("解析出float出错")
				}
				newInstance.Elem().FieldByName(column.Name).SetFloat(d)
				continue
			case "bool":
				d, err := parseBool(content)
				if err != nil {
					log.Println("解析出bool出错")
				}
				newInstance.Elem().FieldByName(column.Name).SetBool(d)
				continue
			case "time", "time.Time":
				var tt reflect.Value
				// 判断是否是字符串
				if IsNum(content) {
					// 这里需要转换成时间
					d, err := strconv.ParseInt(content, 10, 64)
					if err != nil {
						return reflect.Value{}, errors.Wrap(err, "time change to int error")
					}
					tt = reflect.ValueOf(time.Unix(d, 0))
				} else {
					formatTime, err := time.ParseInLocation("2006-01-02 15:04:05", content, time.Local)
					if err != nil {
						return reflect.Value{}, errors.Wrap(err, "time parse location error")
					}
					tt = reflect.ValueOf(formatTime)
				}
				newInstance.Elem().FieldByName(column.Name).Set(tt)
				continue
			}
		}
	}

	return newInstance, nil
}

// id赋值
func (lib *SpAdmin) incrSetValue(data reflect.Value, routerName string, id uint64) reflect.Value {
	cb, _ := lib.config.tableNameGetModelInfo(routerName)
	for _, column := range cb.FieldList.Fields {
		if column.MapName == cb.FieldList.AutoIncrement {
			switch column.Types {
			case "int", "int8", "int16", "int32", "int64":
				data.Elem().FieldByName(column.Name).SetInt(int64(id))
				return data
			case "uint", "uint8", "uint16", "uint32", "uint64":
				data.Elem().FieldByName(column.Name).SetUint(id)
				return data
			}

		}
	}
	return data
}

// 变更用户密码
func (lib *SpAdmin) changeUserPassword(id uint64, password string) error {
	ps, salt := lib.config.passwordSalt(password)
	// 获取表名
	routerName := lib.config.getUserModelTableName()
	model, err := lib.config.tableNameGetModel(routerName)
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	newInstance := reflect.New(t)
	newInstance.Elem().FieldByName("Password").SetString(ps)
	newInstance.Elem().FieldByName("Salt").SetString(salt)
	// 直接变更
	uid, err := lib.config.Engine.Table(routerName).ID(id).Cols("password", "salt").Update(newInstance.Interface())
	if uid == 0 || err != nil {
		return MsgLog(fmt.Sprintf("edit data fail %s id:%d router:%s", err, id, routerName))
	}
	return nil
}

// 注册视图
func (lib *SpAdmin) Register() {
	// $ go get -u github.com/go-bindata/go-bindata/v3/go-bindata
	// $ go-bindata -o bindata.go -pkg simple_admin -prefix "simple_admin_templates" -fs ./simple_admin_templates/...
	// $ go build
	app := lib.config.App
	app.HandleDir("/simple_admin_static", AssetFile())
	app.PartyFunc(lib.config.Prefix, lib.Router)
	// 其他所有操作都重定向
	app.PartyFunc(lib.prefix, func(router router.Party) {
		router.RegisterView(iris.Blocks(AssetFile(), ".html"))
		router.Get("/{root:path}", Index)
	})
	app.UseGlobal(SpiderVisitHistoryMiddleware)
}

func init() {
	log.SetPrefix("[simple_admin] ")
}
