package simple_admin

import (
	"errors"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/imdario/mergo"
	"github.com/kataras/iris/v12"
	"log"
	"strconv"
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
			"user_manage": "user_manage",
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
	//router.Get("/", Index)
	// 登录
	router.Post("/login", Login)
	// 注册
	router.Post("/reg", Reg)
	c := router.Party("/v", CustomJwt.Serve, TokenToUserUidMiddleware)
	// 获取所有表
	c.Get("/get_routers", GetRouters)
	c.Get("/get_routers/{routerName:string}", GetRouterFields)
	// 获取单表列信息
	// todo 接下来开发这部分

	//// 查看
	//c.Get("/{routerName:string}")
	//c.Get("/{routerName:string}/{id:uint64}")
	//// 增加
	//c.Post("/{routerName:string}")
	//// 修改
	//c.Put("/{routerName:string}")
	//// 删除
	//c.Delete("/{routerName:string}/{id:uint64}")
}

// 权限变更
func (lib *SpAdmin) policyChange(userId, path, methods string, add bool) error {
	if add {
		// 先判断权限是否存在
		if lib.casbinEnforcer.HasPolicy(userId, path, methods) {
			return errors.New("policy has exists")
		}
		success, err := lib.casbinEnforcer.AddPolicy(userId, path, methods)
		if err != nil || success == false {
			return errors.New(fmt.Sprintf("add policy fail -> %s %s %s err:%s", userId, path, methods, err))
		}
		return nil
	}
	success, err := lib.casbinEnforcer.RemovePolicy(userId, path, methods)
	if err != nil || success == false {
		return errors.New(fmt.Sprintf("remove policy fail -> %s %s %s err:%s", userId, path, methods, err))
	}
	return nil

}

// 获取权限 根据注册model filterMethods only needs methods data
func (lib *SpAdmin) getAllPolicy(userIdOrRoleName string, filterMethods []string) [][]string {
	policyList := make([][]string, (len(lib.modelTables)+len(lib.sitePolicy))*len(lib.defaultMethods))
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
		return 0, errors.New(fmt.Sprintf("role params not in %s", values))
	}
	ps, salt := lib.config.passwordSalt(password)
	// 获取表名
	tableName := lib.config.getUserModelTableName()
	// 判断用户是否存在
	has, err := lib.config.Engine.Table(tableName).Where("user_name = ?", userName).Exist()
	if has == true {
		return 0, errors.New("user has exist")
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
		return 0, errors.New("insert user fail")
	}

	uid := strconv.FormatInt(userUid, 10)

	// 把用户写入群组
	stats, err := lib.casbinEnforcer.AddRoleForUser(uid, role)
	if stats != true || err != nil {
		return 0, errors.New(fmt.Sprintf("add user to role:%s fail %s", role, err))
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
				return errors.New(fmt.Sprintf("init guest role fail %s", err))
			}
			break
		case "staff":
			// 职员只能看
			rules := lib.getAllPolicy(role, []string{"GET"})
			for _, rule := range rules {
				if rule != nil {
					_, err := lib.casbinEnforcer.AddPermissionForUser(role, rule[1], rule[2])
					if err != nil {
						return errors.New(fmt.Sprintf("init staff role fail %s", err))
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
						return errors.New(fmt.Sprintf("init admin role fail %s", err))
					}
				}
			}
			// 管理员还能进行用户管理
			for _, value := range lib.defaultMethods {
				_, err := lib.casbinEnforcer.AddPermissionForUser(role, lib.sitePolicy["user_manage"], value)
				if err != nil {
					return errors.New(fmt.Sprintf("init admin user manage fail  %s", err))
				}
			}

			break
		}
	}
	// 创建角色继承
	_, err := lib.casbinEnforcer.AddRoleForUser(lib.defaultRole["admin"], lib.defaultRole["staff"])
	if err != nil {
		return errors.New(fmt.Sprintf("role admin has stfall fail  %s", err))
	}
	_, err = lib.casbinEnforcer.AddRoleForUser(lib.defaultRole["staff"], lib.defaultRole["guest"])
	if err != nil {
		return errors.New(fmt.Sprintf("role staff has guest fail %s", err))
	}
	return nil
}

// 分页
func (lib *SpAdmin) Pagination() {

}

// 注册视图
func (lib *SpAdmin) Register() {
	// $ go get -u github.com/go-bindata/go-bindata/...
	// $ go-bindata ./templates/...
	// $ go build
	app := lib.config.App
	app.RegisterView(iris.HTML("./templates", ".html").Binary(Asset, AssetNames))
	app.HandleDir(lib.config.Prefix, "./templates", iris.DirOptions{
		Asset:      Asset,
		AssetInfo:  AssetInfo,
		AssetNames: AssetNames,
		IndexName:  "index.html", // default.
		// If you want to show a list of embedded files when inside a directory without an index file:
		// ShowList:   true,
		// DirList: func(ctx iris.Context, dirName string, f http.File) error {
		// 	// [Optional, custom code to show the html list].
		// }
	})
	app.PartyFunc(lib.config.Prefix, lib.Router)
}
