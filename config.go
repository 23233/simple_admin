package simple_admin

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	xormadapter "github.com/casbin/xorm-adapter"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"log"
	"reflect"
	"strings"
	"time"
	"xorm.io/xorm"
)

type Config struct {
	Name                       string            // 后台显示名称
	Engine                     *xorm.Engine      // xorm engine实例
	App                        *iris.Application // iris实例
	ModelList                  []interface{}     // 模型列表
	UserModel                  interface{}       // 用户模型
	RunSync                    bool              // 是否进行sync
	PageSize                   int               // 每页条数
	AbridgeName                string            // tag的解析名称
	Prefix                     string            // 前缀
	InitAdminUserName          string            // 初始管理员用户名 若存在则跳过
	InitAdminPassword          string            // 初始管理员密码
	UserModelSpecialUniqueName string            // 用户模型唯一名
	CustomAction               []CustomAction    // 自定义action列表
	EnableSpiderWait           bool              // 开启爬虫监听
	spiderModel                interface{}       // 爬虫监听模型
}

// 默认用户模型
type UserModel struct {
	Id       uint64 `xorm:"autoincr pk unique" json:"id"`
	UserName string `xorm:"varchar(60) notnull unique index" json:"username"`
	Password string `xorm:"varchar(100) notnull" json:"password"`
	Salt     string `xorm:"varchar(40) notnull" json:"salt"`
}

// 模型爬虫监听模型
type SpiderHistory struct {
	Id         uint64    `xorm:"autoincr pk unique" json:"id"`
	CreateTime time.Time `xorm:"created index" json:"create_time"`
	Ip         string    `xorm:"varchar(15)" json:"ip"`
	Ua         string    `xorm:"varchar(150)" json:"ua"`
	Page       string    `xorm:"varchar(150)" json:"page"` // 访问路径
}

// 自定义action
type CustomAction struct {
	Name    string        `json:"name"`    // action display name
	Methods string        `json:"methods"` // request run methods
	Valid   interface{}   `json:"valid"`   // request valid struct
	Path    string        `json:"path"`    // request path
	Scope   []interface{} `json:"scope"`   // show where
	Func    func(ctx context.Context)
}

// 默认配置文件
func (config *Config) init() Config {
	return Config{
		Name:                       "simpleAdmin",
		UserModel:                  new(UserModel),
		RunSync:                    true,
		PageSize:                   20,
		Prefix:                     "/admin",
		InitAdminUserName:          "admin",
		InitAdminPassword:          "iris_best",
		UserModelSpecialUniqueName: "simple_admin_user_model",
		AbridgeName:                "sp",
		EnableSpiderWait:           true,
		spiderModel:                new(SpiderHistory),
	}
}

// 验证配置文件
func (config *Config) valid() error {
	if config.Engine == nil {
		return MsgLog("please check config , engine is empty")
	}
	if config.App == nil {
		return MsgLog("please check config , app(iris instance application)  is empty")
	}
	if reflect.DeepEqual(config.UserModel, new(UserModel)) == false {
		log.Printf("custom user model warning : 1.must has username password salt id fields 2.username must be unique 3.id must be autoincr fields")
	}
	if len(config.ModelList) < 1 {
		return MsgLog("please check config , modelList is empty ")
	}
	if len(config.Prefix) < 1 {
		return MsgLog("please check config , prefix is required")
	}
	if config.Prefix[0] != '/' {
		return MsgLog("please check config , prefix must start with / ")
	}
	if len(config.CustomAction) >= 1 {
		for _, action := range config.CustomAction {
			if len(action.Scope) < 1 {
				return MsgLog("please check config, custom action scope is required")
			}
			if reflect.ValueOf(action.Valid).IsNil() || len(action.Name) < 1 {
				return MsgLog("please check config, custom action all fields is required")
			}
			if len(action.Path) >= 1 {
				if strings.HasPrefix(action.Path, "/") == false {
					return MsgLog("custom action path must be use / start prefix")
				}
			}
		}
	}
	return nil
}

// 合并验证自定义action
func (config *Config) validAction() {
	var result []CustomAction
	for _, action := range config.CustomAction {
		var d CustomAction
		if len(action.Methods) < 1 {
			d.Methods = "POST"
		} else {
			d.Methods = action.Methods
		}
		if len(action.Path) < 1 {
			d.Path = "p_" + RandStringBytes(6)
		} else {
			d.Path = action.Path
		}
		d.Name = action.Name
		d.Valid = action.Valid

		d.Scope = action.Scope
		d.Func = action.Func
		result = append(result, d)
	}
	config.CustomAction = result
}

// 配置文件初始化权限
func (config *Config) initCasbin() (*casbin.Enforcer, error) {
	rbac :=
		`
		[request_definition]
		r = sub, obj, act
		
		[policy_definition]
		p = sub, obj, act
		
		[role_definition]
		g = _, _
		
		[policy_effect]
		e = some(where (p.eft == allow))
		
		[matchers]
		m = g(r.sub, p.sub) && keyMatch3(r.obj, p.obj) && (r.act == p.act || p.act == "*")
		`
	m, err := model.NewModelFromString(rbac)
	if err != nil {
		log.Fatalf("load model from string demo error %s", err)
		return nil, err
	}
	adapter, err := xormadapter.NewAdapterByEngine(config.Engine)
	if err != nil {
		log.Fatalf("init by engine error %s", err)
		return nil, err
	}
	Enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		log.Fatalf("init to new enforcer error %s", err)
		return nil, err

	}
	return Enforcer, nil
}

// 进行SYNC
func (config *Config) runSync() error {
	if config.RunSync {
		err := config.Engine.Sync2(config.ModelList...)
		if err != nil {
			return err
		}
	}
	err := config.Engine.Sync2(config.UserModel)
	return err
}

// 通过表名匹配是否有自定义action
func (config *Config) tableNameCustomActionScopeMatch(routerName string) []CustomActionResp {
	result := make([]CustomActionResp, 0)
	for _, action := range config.CustomAction {
		for _, scope := range action.Scope {
			tableName := config.Engine.TableName(scope)
			if tableName == routerName {
				var d CustomActionResp
				d.Path = action.Path
				d.Methods = action.Methods
				d.Name = action.Name
				values := config.tableNameGetNestedStructMaps(reflect.TypeOf(action.Valid))
				d.Fields = values
				result = append(result, d)
			}
		}
	}
	return result
}

// 模型表名序列生成
func (config *Config) generateTables() []string {
	tables := make([]string, 0, len(config.ModelList))
	for _, item := range config.ModelList {
		tableName := config.Engine.TableName(item)
		tables = append(tables, tableName)
	}
	return tables
}

// 通过模型名获取模型信息
func (config *Config) tableNameToFieldAndTypes(tableName string) (map[string]string, error) {
	for _, item := range config.ModelList {
		if config.Engine.TableName(item) == tableName {
			t := reflect.TypeOf(item)
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			fieldNum := t.NumField()
			result := make(map[string]string, fieldNum)
			for i := 0; i < fieldNum; i++ {
				f := t.Field(i)
				n := t.Field(i).Name
				//// 判断是否为自增主键
				//x := f.Tag.Get("xorm")
				//if n == "Id" && len(x) < 1 || strings.Index(x, "autoincr") >= 0 {
				//	result["autoincr"] = n
				//}
				result[n] = f.Type.String()
			}
			return result, nil
		}
	}
	return nil, MsgLog(fmt.Sprintf("not find this table %s", tableName))
}

// 通过模型反射模型信息
func (config *Config) tableNameReflectFieldsAndTypes(tableName string) (TableFieldsResp, error) {
	for _, item := range config.ModelList {
		if config.Engine.TableName(item) == tableName {
			modelInfo, err := NowSpAdmin.config.Engine.TableInfo(item)
			if err != nil {
				return TableFieldsResp{}, nil
			}
			var resp TableFieldsResp
			// 获取三要素
			values := config.tableNameGetNestedStructMaps(reflect.TypeOf(item))
			resp.Fields = values
			resp.Autoincr = modelInfo.AutoIncrement
			resp.Version = modelInfo.Version
			resp.Deleted = modelInfo.Deleted
			resp.Updated = modelInfo.Updated
			return resp, nil
		}
	}
	return TableFieldsResp{}, MsgLog(fmt.Sprintf("not find this table %s", tableName))

}

// 通过模型名获取所有列信息 名称 类型 xorm tag validator comment
func (config *Config) tableNameGetNestedStructMaps(r reflect.Type) []structInfo {
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	if r.Kind() != reflect.Struct {
		return nil
	}
	v := reflect.New(r).Elem()
	result := make([]structInfo, 0)
	for i := 0; i < r.NumField(); i++ {
		field := r.Field(i)
		v := reflect.Indirect(v).FieldByName(field.Name)
		fieldValue := v.Interface()
		var d structInfo

		switch fieldValue.(type) {
		case time.Time, time.Duration:
			d.Name = field.Name
			d.Types = field.Type.String()
			d.XormTags = field.Tag.Get("xorm")
			d.SpTags = field.Tag.Get(config.AbridgeName)
			d.ValidateTags = field.Tag.Get("validate")
			d.CommentTags = field.Tag.Get("comment")
			d.AttrTags = field.Tag.Get("attr")
			d.MapName = config.Engine.GetColumnMapper().Obj2Table(field.Name)
			result = append(result, d)
			continue
		}
		if field.Type.Kind() == reflect.Struct {
			values := config.tableNameGetNestedStructMaps(field.Type)
			result = append(result, values...)
			continue
		}
		d.Name = field.Name
		d.Types = field.Type.String()
		d.MapName = config.Engine.GetColumnMapper().Obj2Table(field.Name)
		d.XormTags = field.Tag.Get("xorm")
		d.SpTags = field.Tag.Get(config.AbridgeName)
		d.CommentTags = field.Tag.Get("comment")
		d.AttrTags = field.Tag.Get("attr")
		d.ValidateTags = field.Tag.Get("validate")
		result = append(result, d)
	}
	return result
}

// 通过模型名获取实例
func (config *Config) tableNameGetModel(tableName string) (interface{}, error) {
	for _, item := range config.ModelList {
		if config.Engine.TableName(item) == tableName {
			return item, nil
		}
	}
	return nil, MsgLog("not find table")
}

// 获取用户表
func (config *Config) getUserModelTableName() string {
	tableName := config.Engine.TableName(config.UserModel)
	return tableName
}

// 密码加密
func (config *Config) passwordSalt(password string) (string, string) {
	salt := RandStringBytes(4)
	m5 := md5.New()
	m5.Write([]byte(password))
	m5.Write([]byte(salt))
	st := m5.Sum(nil)
	ps := hex.EncodeToString(st)
	return ps, salt
}

// 密码比较
func (config *Config) validPassword(password, salt, m5 string) bool {
	r := md5.New()
	r.Write([]byte(password))
	r.Write([]byte(salt))
	st := r.Sum(nil)
	ps := hex.EncodeToString(st)

	return ps == m5

}
