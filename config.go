package simple_admin

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	xormadapter "github.com/casbin/xorm-adapter"
	"github.com/kataras/iris/v12"
	"log"
	"reflect"
	"xorm.io/xorm"
)

type Config struct {
	Name                       string            // 后台显示名称
	Engine                     *xorm.Engine      // xorm engine实例
	App                        *iris.Application // iris实例
	ModelList                  []interface{}     // 模型列表
	UserModel                  interface{}       // 用户模型
	RunSync                    bool              // 是否进行sync
	EnableReg                  bool              // 是否启用注册
	PageSize                   int
	Prefix                     string
	InitAdminUserName          string
	InitAdminPassword          string
	UserModelSpecialUniqueName string
}

type UserModel struct {
	Id       uint64 `xorm:"autoincr pk unique" json:"id"`
	UserName string `xorm:"varchar(60) notnull unique index" json:"username"`
	Password string `xorm:"varchar(100) notnull" json:"password"`
	Salt     string `xorm:"varchar(40) notnull" json:"salt"`
}

// 默认配置文件
func (config *Config) init() Config {
	return Config{
		Name:                       "simpleAdmin",
		UserModel:                  new(UserModel),
		RunSync:                    true,
		EnableReg:                  true,
		PageSize:                   20,
		Prefix:                     "/admin",
		InitAdminUserName:          "admin",
		InitAdminPassword:          "iris_best",
		UserModelSpecialUniqueName: "simple_admin_user_model",
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
	return nil
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
func (config *Config) tableNameReflectFieldsAndTypes(tableName string) (map[string]string, error) {
	for _, item := range config.ModelList {
		if config.Engine.TableName(item) == tableName {
			modelInfo, err := NowSpAdmin.config.Engine.TableInfo(item)
			if err != nil {
				return nil, err
			}
			result := make(map[string]string, len(modelInfo.ColumnsSeq()))
			t := reflect.TypeOf(item)
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			for _, column := range modelInfo.Columns() {
				if column.Name == modelInfo.AutoIncrement {
					result["autoincr"] = column.Name
					continue
				}
				f, has := t.FieldByName(column.FieldName)
				if has == false {
					continue
				}
				result[column.Name] = f.Type.String()
			}
			return result, nil
		}
	}
	return nil, MsgLog(fmt.Sprintf("not find this table %s", tableName))

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
