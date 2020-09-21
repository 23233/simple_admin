package simple_admin

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	xormadapter "github.com/casbin/xorm-adapter"
	"github.com/pkg/errors"
	"log"
	"reflect"
	"strings"
	"time"
)

// 默认配置文件
func (config *Config) initConfig() Config {
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
		EnableSpiderWatch:          true,
		SpiderMatchList:            []string{"spider", "Spider", "bot", "Bot", "crawler", "trident", "Trident", "Slurp", "craw"},
	}
}

// 表内信息扫描
func (config *Config) scanTableInfo() {
	var result []TableInfoList
	// 判断是否启用爬虫监测
	if config.EnableSpiderWatch {
		config.ModelList = append(config.ModelList, new(SpiderHistory))
	}
	// 把用户模型合并到模型表格中
	config.ModelList = append(config.ModelList, config.UserModel)
	for _, item := range config.ModelList {
		name := config.Engine.TableName(item)
		cb, err := config.tableNameReflectFieldsAndTypes(name)
		if err != nil {
			panic(errors.Wrap(err, fmt.Sprintf("初始化扫描表:%s信息出错", name)))
		}
		var d TableInfoList
		if processor, ok := item.(SpTableNameProcess); ok {
			d.RemarkName = processor.Remark()
		}
		d.RouterName = name
		d.FieldList = cb
		d.Actions = config.validAction(item)
		result = append(result, d)
	}
	config.modelInfoList = result
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
		config.Prefix = "/" + config.Prefix
	}
	return nil
}

// 合并验证自定义action
func (config *Config) validAction(item interface{}) []CustomAction {
	var resultList []CustomAction
	typ := reflect.TypeOf(item)
	vtp := reflect.ValueOf(item)
	for i := 0; i < vtp.NumMethod(); i++ {
		m := vtp.Method(i)
		tm := typ.Method(i)
		if !strings.HasPrefix(tm.Name, "SpAction") {
			continue
		}
		// 判断返回值是否正确
		if tm.Type.NumOut() < 1 {
			continue
		}
		out := tm.Type.Out(0)
		if out.Kind() != reflect.Struct {
			continue
		}

		result := m.Call(nil)[0]

		actionBase := reflect.TypeOf(new(CustomAction))
		if actionBase.Kind() == reflect.Ptr {
			actionBase = actionBase.Elem()
		}

		var d = reflect.Indirect(reflect.ValueOf(new(CustomAction)))
		for p := 0; p < actionBase.NumField(); p++ {
			if reflect.Indirect(d.Field(p)).CanInterface() {
				var n = actionBase.Field(p).Name
				reflect.TypeOf(actionBase.Field(p))
				d.FieldByName(n).Set(result.FieldByName(n))
			}
		}
		var r = d.Interface().(CustomAction)
		if reflect.Indirect(reflect.ValueOf(r.Func)).IsNil() {
			name := config.Engine.TableName(item)
			log.Printf("[%s]自定义%s执行方法验证错误", name, tm.Name)
			continue
		}
		if len(r.Methods) < 1 {
			r.Methods = "POST"
		}
		if len(r.Path) < 1 {
			r.Path = "p_" + RandStringBytes(6)
		}
		r.Scope = item
		r.hasValid = !reflect.ValueOf(r.Valid).IsNil()
		resultList = append(resultList, r)
	}

	return resultList
}

// 配置文件初始化权限
func (config *Config) initCasBin() (*casbin.Enforcer, error) {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", `g(r.sub, p.sub) && keyMatch3(r.obj, p.obj) && (r.act == p.act || p.act == "*")`)
	adapter, err := xormadapter.NewAdapterByEngine(config.Engine)
	if err != nil {
		log.Fatalf("initConfig by engine error %s", err)
		return nil, err
	}
	Enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		log.Fatalf("initConfig to new enforcer error %s", err)
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
	for _, m := range config.modelInfoList {
		if routerName == m.RouterName {
			for _, action := range m.Actions {
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
	tables := make([]string, 0, len(config.modelInfoList))
	for _, item := range config.modelInfoList {
		tables = append(tables, item.RouterName)
	}
	return tables
}

// 通过模型反射模型信息
func (config *Config) tableNameReflectFieldsAndTypes(tableName string) (TableFieldsResp, error) {
	for _, item := range config.ModelList {
		if config.Engine.TableName(item) == tableName {
			modelInfo, err := config.Engine.TableInfo(item)
			if err != nil {
				return TableFieldsResp{}, nil
			}
			var resp TableFieldsResp
			// 获取三要素
			values := config.tableNameGetNestedStructMaps(reflect.TypeOf(item))
			resp.Fields = values
			resp.AutoIncrement = modelInfo.AutoIncrement
			resp.Version = modelInfo.Version
			resp.Deleted = modelInfo.Deleted
			resp.Created = modelInfo.Created
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

// 通过模型名获取模型信息
func (config *Config) tableNameGetModelInfo(tableName string) (TableInfoList, error) {
	for _, l := range config.modelInfoList {
		if l.RouterName == tableName {
			return l, nil
		}
	}
	return TableInfoList{}, errors.New("not found model")
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
