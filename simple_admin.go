package simple_admin

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/imdario/mergo"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	NowSpAdmin *SpAdmin
)

type SpAdmin struct {
	config         Config
	casbinEnforcer *casbin.Enforcer
	modelTables    []string
	defaultMethods map[string]string // 默认权限方法
	defaultRole    map[string]string // 默认角色
	sitePolicy     map[string]string
}

type Policy struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type PagResult struct {
	All      int64               `json:"all"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Data     []map[string]string `json:"data"`
}

func (lib *SpAdmin) errorLog(err error, msg string) error {
	return errors.Wrap(err, msg)
}

func New(c Config) (*SpAdmin, error) {
	// 合并配置文件
	init := new(Config).init()
	if err := mergo.Map(&c, init); err != nil {
		return nil, err
	}
	if err := c.valid(); err != nil {
		return nil, err
	}

	// 进行初始化权限系统
	enforcer, err := c.initCasbin()
	if err != nil {
		return nil, err
	}

	// 进行sync操作
	if err := c.runSync(); err != nil {
		return nil, err
	}

	// 把用户模型合并到模型表格中
	c.ModelList = append(c.ModelList, c.UserModel)

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
		log.Printf("init admin user fail: %s", err.Error())
	}

	return NowSpAdmin, nil
}

// 在这里注册路由
func (lib *SpAdmin) Router(router iris.Party) {
	// 首页
	router.Get("/", Index)
	// 登录
	router.Post("/login", Login)
	// 注册
	router.Post("/reg", Reg)
	router.Get("/config", Configuration)
	c := router.Party("/v", CustomJwt.Serve, TokenToUserUidMiddleware)
	// 获取当前用户
	c.Get("/get_current_user", GetCurrentUser)
	// 变更用户密码
	c.Post("/change_password", ChangeUserPassword)
	// 获取所有表
	c.Get("/get_routers", GetRouters)
	// 获取单表列信息
	c.Get("/get_routers_fields/{routerName:string}", PolicyValidMiddleware, GetRouterFields)
	// 查看
	c.Get("/{routerName:string}", PolicyValidMiddleware, GetRouterData)
	c.Get("/{routerName:string}/{id:uint64}", PolicyValidMiddleware, GetRouterSingleData)
	// 增加
	c.Post("/{routerName:string}", PolicyValidMiddleware, AddRouterData)
	// 修改
	c.Put("/{routerName:string}/{id:uint64}", PolicyValidMiddleware, EditRouterData)
	// 删除 delete模式在某些匹配时候有问题
	c.Post("/{routerName:string}/delete", PolicyValidMiddleware, RemoveRouterData)
	// 权限相关
	c.Post("/change_user_role", PolicyRequireAdminMiddleware, ChangeUserRoles)

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
func (lib *SpAdmin) getAllPolicy(userIdOrRoleName string, filterMethods []string) [][]string {
	policyList := make([][]string, 0, (len(lib.modelTables)+len(lib.sitePolicy))*len(lib.defaultMethods))
	var d []string
	for _, v := range lib.sitePolicy {
		d = append(d, v)
	}
	full := append(lib.modelTables, d...)
	for _, item := range full {
		if len(item) >= 1 {
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
				return MsgLog(fmt.Sprintf("init guest role fail %s", err))
			}
			break
		case "staff":
			// 职员只能看
			rules := lib.getAllPolicy(role, []string{"GET"})
			for _, rule := range rules {
				if rule != nil {
					_, err := lib.casbinEnforcer.AddPermissionForUser(role, rule[1], rule[2])
					if err != nil {
						return MsgLog(fmt.Sprintf("init staff role fail %s", err))
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
						return MsgLog(fmt.Sprintf("init admin role fail %s", err))
					}
				}
			}
			// 管理员还能进行用户管理
			for _, value := range lib.defaultMethods {
				_, err := lib.casbinEnforcer.AddPermissionForUser(role, lib.sitePolicy["user_manage"], value)
				if err != nil {
					return MsgLog(fmt.Sprintf("init admin user manage fail  %s", err))
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
	pageSize := lib.config.PageSize
	start := (page - 1) * pageSize
	offset := pageSize
	end := page*pageSize + offset
	// 先获取总数量
	allCount, err := lib.config.Engine.Table(routerName).Count()
	if err != nil {
		return p, err
	}

	data, err := lib.config.Engine.Table(routerName).And("id between ? and ?", start, end).Limit(pageSize).QueryString()
	if err != nil {
		return p, err
	}

	// 如果是用户表 还需要返回当前的权限
	// 只有admin 才能请求到用户表
	if routerName == lib.config.getUserModelTableName() {
		model, _ := lib.config.tableNameGetModel(routerName)
		modelInfo, err := lib.config.Engine.TableInfo(model)
		if err != nil {
			return p, err
		}
		for i, d := range data {
			for k, v := range d {
				if k == modelInfo.AutoIncrement {
					roles, err := lib.casbinEnforcer.GetImplicitRolesForUser(v)
					if err != nil {
						return p, err
					}
					data[i]["roles"] = strings.Join(roles, ",")
					break
				}
			}
		}
	}

	p.PageSize = pageSize
	p.Page = page
	p.All = allCount
	p.Data = data
	return p, nil

}

// 单条数据获取
func (lib *SpAdmin) SingleData(routerName string, id uint64) (map[string]string, error) {
	var valuesMap = make(map[string]string)
	has, err := lib.config.Engine.Table(routerName).Where("id = ?", id).Get(&valuesMap)
	if err != nil {
		return valuesMap, err
	}
	if has == false {
		return valuesMap, MsgLog("not find data")
	}
	return valuesMap, nil
}

// 新增数据
func (lib *SpAdmin) addData(routerName string, data reflect.Value) error {
	uid, err := lib.config.Engine.Table(routerName).InsertOne(data.Interface())
	if uid == 0 || err != nil {
		return MsgLog(fmt.Sprintf("insert data fail %s", err))
	}
	// 获取
	return nil
}

// 数据修改
func (lib *SpAdmin) editData(routerName string, id uint64, data reflect.Value) error {
	// 默认只更新非空和非0的字段 xorm的规则
	// 所以这里启动全量更新 传入数据必须为全量
	uid, err := lib.config.Engine.Table(routerName).ID(id).AllCols().Update(data.Interface())
	if uid == 0 || err != nil {
		return MsgLog(fmt.Sprintf("edit data fail %s id:%d router:%s", err, id, routerName))
	}
	// 获取
	return nil
}

// 数据删除
func (lib *SpAdmin) deleteData(routerName string, id uint64) error {
	affected, err := lib.config.Engine.Exec(fmt.Sprintf("delete from %s where id = ?", routerName), id)
	if err != nil {
		return err
	}
	obj, err := affected.RowsAffected()

	if obj < 1 {
		return MsgLog("delete data fail ")
	}
	return nil
}

// 批量数据删除
func (lib *SpAdmin) bulkDeleteData(routerName string, ids string) error {
	affected, err := lib.config.Engine.Exec(fmt.Sprintf("delete from %s where id in (%s)", routerName, ids))
	if err != nil {
		return err
	}
	obj, err := affected.RowsAffected()

	if obj < 1 {
		return MsgLog("delete data fail ")
	}
	return nil
}

// 判断数据是否存在
func (lib *SpAdmin) dataExists(routerName string, id uint64) (bool, error) {
	return lib.config.Engine.Table(routerName).Where("id = ?", id).Exist()
}

// 对应关系获取
func (lib *SpAdmin) getCtxValues(routerName string, ctx iris.Context) (reflect.Value, error) {
	// 先获取到字段信息
	model, err := lib.config.tableNameGetModel(routerName)
	// 拿到字段对应类型
	fieldTypes, err := lib.config.tableNameToFieldAndTypes(routerName)
	if err != nil {
		return reflect.Value{}, err
	}
	modelInfo, err := lib.config.Engine.TableInfo(model)
	if err != nil {
		return reflect.Value{}, err
	}
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	newInstance := reflect.New(t)

	for _, column := range modelInfo.Columns() {
		if column.Name != modelInfo.AutoIncrement {
			// 判断类型进行赋值
			f := fieldTypes[column.FieldName]
			switch f {
			case "string":
				d := ctx.PostValue(column.Name)
				newInstance.Elem().FieldByName(column.FieldName).SetString(d)
				continue
			case "int", "int8", "int16", "int32", "int64", "time.Duration":
				d, err := ctx.PostValueInt(column.Name)
				if err != nil {
					return reflect.Value{}, err
				}
				newInstance.Elem().FieldByName(column.FieldName).SetInt(int64(d))
				continue
			case "uint", "uint8", "uint16", "uint32", "uint64":
				d, err := ctx.PostValueInt(column.Name)
				if err != nil {
					ctx.StatusCode(iris.StatusBadRequest)
					_, _ = ctx.JSON(iris.Map{
						"detail": err.Error(),
					})
					return reflect.Value{}, err
				}
				newInstance.Elem().FieldByName(column.FieldName).SetUint(uint64(d))
				continue
			case "float32", "float64":
				d, err := ctx.PostValueFloat64(column.Name)
				if err != nil {
					return reflect.Value{}, err
				}
				newInstance.Elem().FieldByName(column.FieldName).SetFloat(d)
				continue
			case "bool":
				d, err := ctx.PostValueBool(column.Name)
				if err != nil {
					return reflect.Value{}, err
				}
				newInstance.Elem().FieldByName(column.FieldName).SetBool(d)
				continue
			case "time", "time.Time":
				d := ctx.PostValue(column.Name)
				if len(d) < 1 {
					return reflect.Value{}, err
				}
				var tt reflect.Value
				// 判断是否是字符串
				if IsNum(d) {
					// 这里需要转换成时间
					d, err := strconv.ParseInt(d, 10, 64)
					if err != nil {
						return reflect.Value{}, err
					}
					tt = reflect.ValueOf(time.Unix(d, 0))
				} else {
					formatTime, err := time.ParseInLocation("2006-01-02 15:04:05", d, time.Local)
					if err != nil {
						return reflect.Value{}, err
					}
					tt = reflect.ValueOf(formatTime)
				}

				newInstance.Elem().FieldByName(column.FieldName).Set(tt)
				continue
			}
		}
	}
	return newInstance, nil
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
	// $ go get -u github.com/go-bindata/go-bindata/...
	// $ go-bindata ./templates/...
	// $ go build
	app := lib.config.App
	app.RegisterView(iris.HTML("./templates", ".html").Binary(Asset, AssetNames))
	app.HandleDir("/simple_admin_static", "./templates", iris.DirOptions{
		Asset:      Asset,
		AssetInfo:  AssetInfo,
		AssetNames: AssetNames,
		//IndexName:  "index.html", // default.
		Gzip: true,
		// If you want to show a list of embedded files when inside a directory without an index file:
		//ShowList: true,
		// DirList: func(ctx iris.Context, dirName string, f http.File) error {
		// 	// [Optional, custom code to show the html list].
		// }
	})
	app.PartyFunc(lib.config.Prefix, lib.Router)
}
