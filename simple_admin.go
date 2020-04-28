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

	// 生成表名列表
	modelTables := c.generateTables()

	NowSpAdmin = &SpAdmin{
		config:         c,
		casbinEnforcer: enforcer,
		modelTables:    modelTables,
	}
	// 进行视图注册绑定
	NowSpAdmin.Register()

	// 进行初始化管理员操作
	if c.EnableInitAdmin {
		// 初始化管理员
		_, err := NowSpAdmin.addUser(c.InitAdminUserName, c.InitAdminPassword)
		if err != nil {
			log.Printf("init admin user fail: %s", err.Error())
		}
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
	// todo 接下来开发这部分
	//c := router.Party("/v",CustomJwt.Serve, TokenToUserUidMiddleware)
	//// 获取所有表
	//c.Get("/get_routers")
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

// 新增登录网站权限
func (lib *SpAdmin) addLoginSitePermission(userId string) error {
	return lib.policyChange(userId, "login_site", "POST", true)
}

// 是否拥有登录权限
func (lib *SpAdmin) hasLoginPolicy(userId string) bool {
	return lib.policyHas(userId, "login_site", "POST")
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

// 权限校验
func (lib *SpAdmin) policyHas(userId, path, methods string) bool {
	return lib.casbinEnforcer.HasPolicy(userId, path, methods)
}

// 新建用户
func (lib *SpAdmin) addUser(userName, password string) (int64, error) {
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
	aff, err := success.LastInsertId()
	if aff == 0 || err != nil {
		return 0, errors.New("insert user fail")
	}

	// 写入登录权限
	err = lib.addLoginSitePermission(strconv.FormatInt(aff, 10))
	if err != nil {
		return 0, err
	}
	return aff, nil
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
