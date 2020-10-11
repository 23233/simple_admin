package simple_admin

import (
	"github.com/kataras/iris/v12"
	"time"
	"xorm.io/xorm"
)

type Config struct {
	Name                       string            // 后台显示名称
	Engine                     *xorm.Engine      // xorm engine实例
	App                        *iris.Application // iris实例
	ModelList                  []interface{}     // 模型列表
	modelInfoList              []TableInfoList   // 表信息列表
	UserModel                  interface{}       // 用户模型
	RunSync                    bool              // 是否进行sync
	PageSize                   int               // 每页条数
	AbridgeName                string            // tag的解析名称
	Prefix                     string            // 前缀
	InitAdminUserName          string            // 初始管理员用户名 若存在则跳过
	InitAdminPassword          string            // 初始管理员密码
	UserModelSpecialUniqueName string            // 用户模型唯一名
	CustomActions              []CustomAction    // 自定义action列表
	EnableSpiderWatch          bool              // 开启爬虫监听
	SpiderMatchList            []string          // 爬虫ua匹配列表
	SpiderSkipList             []string          // 爬虫匹配忽略
}

// 默认用户模型
type UserModel struct {
	Id       uint64 `xorm:"autoincr pk unique" json:"id"`
	UserName string `xorm:"varchar(60) notnull unique index" comment:"用户名" json:"username"`
	Password string `xorm:"varchar(100) notnull" comment:"密码" json:"password"`
	Salt     string `xorm:"varchar(40) notnull" comment:"加密salt" json:"salt"`
}

// 模型爬虫监听模型
type SpiderHistory struct {
	Id         uint64    `xorm:"autoincr pk unique" json:"id"`
	CreateTime time.Time `xorm:"created index" json:"create_time" comment:"创建时间"`
	Ip         string    `xorm:"varchar(15)" json:"ip" comment:"ip地址"`
	Match      string    `xorm:"varchar(40)" json:"match" comment:"爬虫名"`
	Ua         string    `xorm:"varchar(150)" json:"ua" comment:"ua"`
	Page       string    `xorm:"varchar(150)" json:"page" comment:"访问路径"` // 访问路径
}

func (c *SpiderHistory) SpAlias() string {
	return "爬虫记录"
}

// 仪表台模型
type DashBoardScreen struct {
	Id           uint64 `xorm:"autoincr pk unique" json:"id"`
	Name         string `xorm:"varchar(45) notnull" comment:"仪表台名称" json:"name"`
	Priority     uint64 `xorm:"notnull" comment:"优先级" json:"priority"`
	IsDefault    bool   `xorm:"notnull" comment:"默认" json:"is_default"`
	CreateUserId uint64 `xorm:"notnull index" comment:"用户ID" json:"create_user_id"`
}

func (c *DashBoardScreen) SpAlias() string {
	return "仪表台屏幕"
}

type DashBoard struct {
	Id         uint64 `xorm:"autoincr pk unique" json:"id"`
	ScreenId   uint64 `sp:"fk('DashBoardScreen') index" comment:"屏幕ID" json:"screen_id" `
	Name       string `xorm:"varchar(45) notnull" comment:"名称" json:"name"`
	ChartType  string `xorm:"varchar(40) notnull" comment:"图表类型" json:"chart_type"`
	DataSource string `xorm:"text notnull" comment:"数据源" json:"data_source"`
	Config     string `xorm:"text notnull" comment:"配置" json:"config"`
}

func (c *DashBoard) SpAlias() string {
	return "仪表台内容"
}

// 自定义action
type CustomAction struct {
	Name     string      `json:"name"`    // action display name
	Methods  string      `json:"methods"` // request run methods
	Valid    interface{} `json:"valid"`   // request valid struct
	Path     string      `json:"path"`    // request path
	Scope    interface{} `json:"scope"`   // show where
	Func     func(ctx iris.Context)
	hasValid bool
}

// 表信息存储
type TableInfoList struct {
	RouterName string          `json:"router_name"`
	RemarkName string          `json:"remark_name"`
	FieldList  TableFieldsResp `json:"field_list"`
	Actions    []CustomAction  `json:"actions"`
}

// 分页结果
type PagResult struct {
	All      int64               `json:"all"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Data     []map[string]string `json:"data"`
}
