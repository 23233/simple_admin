package main

import (
	"github.com/23233/simple_admin"
	"github.com/23233/simple_admin/_examples/database"
	"github.com/23233/simple_admin/_examples/model"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
)

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

	engine := database.Engine
	modelList := []interface{}{
		new(model.TestModelA),
		new(model.TestModelB),
		new(model.ComplexModelC),
		new(model.ComplexModelD),
		new(model.TestStructComplexModel),
	}

	_, err := simple_admin.New(simple_admin.Config{
		Engine:    engine,
		App:       app,
		ModelList: modelList,
		Name:      "测试sync",
		RunSync:   true,
		Prefix:    "/admin",
	})
	if err != nil {
		panic(err)
	}

	//more RegisterView
	tmpl := iris.HTML("_examples/templates", ".html").Layout("layout.html")
	tmpl.Reload(true) // reload templates on each request (development mode)
	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		_ = ctx.View("index.html")
	})

	_ = app.Listen(":8080")

}
