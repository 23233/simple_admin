package main

import (
	"github.com/23233/simple_admin"
	"github.com/23233/simple_admin/_examples/model"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	_ "github.com/mattn/go-sqlite3"
	"time"
	"xorm.io/xorm"
)

var Engine *xorm.Engine

func init() {
	// database 连接器
	var err error

	Engine, err = xorm.NewEngine("sqlite3", "./simple.db")

	if err != nil {
		println(err.Error())
		return
	}
	//Engine.SetLogger()
	Engine.ShowSQL(true)
	//Engine.ShowExecTime(true)
	err = Engine.Ping()
	if err != nil {
		panic(err)
	}

	// timezone时区
	Engine.TZLocation, _ = time.LoadLocation("Asia/Shanghai")
}

func main() {
	app := iris.New()

	app.Logger().SetLevel("debug")

	customLogger := logger.New(logger.Config{
		// Status displays status code
		Status: true,
		// IP displays request's remote address
		IP: true,
		// Method displays the http method
		Method: true,
		// Path displays the request path
		Path: true,
		// Query appends the url query to the Path.
		Query: true,

		// Columns: true,

		// if !empty then its contents derives from `ctx.Values().Get("logger_message")
		// will be added to the logs.
		MessageContextKeys: []string{"logger_message"},

		// if !empty then its contents derives from `ctx.GetHeader("User-Agent")
		MessageHeaderKeys: []string{"User-Agent"},
	})
	app.Use(customLogger)
	app.Use(recover.New())

	modelList := []interface{}{
		new(model.TestModelA),
		new(model.TestModelB),
		new(model.ComplexModelC),
		new(model.ComplexModelD),
		new(model.TestStructComplexModel),
	}

	//more RegisterView
	tmpl := iris.Blocks("_examples/templates", ".html")
	app.RegisterView(tmpl)

	_, err := simple_admin.New(simple_admin.Config{
		Engine:    Engine,
		App:       app,
		ModelList: modelList,
		Name:      "测试sync",
	})
	if err != nil {
		panic(err)
	}

	app.Get("/", func(ctx iris.Context) {
		_ = ctx.View("index")
	})

	_ = app.Listen(":8080")

}
