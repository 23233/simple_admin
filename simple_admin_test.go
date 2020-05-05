package simple_admin_test

import (
	"github.com/23233/simple_admin/v1"
	"github.com/23233/simple_admin/v1/_examples/database"
	"github.com/kataras/iris/v12"
	"testing"
)

type TestModelA struct {
	Id   uint64 `xorm:"autoincr pk unique" json:"id"`
	Name string `xorm:"varchar(20)"`
}

type TestModelB struct {
	Id   uint64 `xorm:"autoincr pk unique" json:"id"`
	Desc string `xorm:"varchar(60)"`
}

type TestUserModel struct {
	Id       uint64 `xorm:"autoincr pk unique" json:"id"`
	UserName string `xorm:"varchar(60) notnull" json:"user_name"`
	Password string `xorm:"varchar(100) notnull" json:"password"`
	Salt     string `xorm:"varchar(40) notnull" json:"salt"`
	Niubi    string `xorm:"varchar(30)"`
}

func TestNew(t *testing.T) {
	t.Parallel()
	app := iris.New()
	engine := database.Engine
	modelList := []interface{}{new(TestModelA), new(TestModelB)}
	// 测试不传或者乱传参数
	//lib, err := simple_admin.New(simple_admin.Config{
	//	Engine:    nil,
	//	ModelList: nil,
	//})
	//if err != nil {
	//	t.Error(err)
	//}
	// 测试进行sync
	lib, err := simple_admin.New(simple_admin.Config{
		Name:      "测试sync",
		Engine:    engine,
		ModelList: modelList,
		RunSync:   true,
	})
	if err != nil {
		t.Error(err)
	}
	//// 测试进行自定义用户模型
	//lib, err = simple_admin.New(simple_admin.Config{
	//	Name:      "测试sync",
	//	Engine:    engine,
	//	ModelList: modelList,
	//	RunSync:   false,
	//	UserModel: new(TestUserModel),
	//	EnableReg: false,
	//})
	//if err != nil {
	//	t.Error(err)
	//}
	app.PartyFunc("/admin", lib.Router)
}
