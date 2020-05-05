package main

import (
	"github.com/23233/simple_admin"
	"github.com/23233/simple_admin/_examples/database"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
	"github.com/kataras/iris/v12/middleware/recover"
	"time"
)

type TestModelA struct {
	Id   uint64 `xorm:"autoincr pk unique" json:"id"`
	Name string `xorm:"varchar(20)"`
}

type TestModelB struct {
	Id   uint64 `xorm:"autoincr pk unique" json:"id"`
	Desc string `xorm:"varchar(60)"`
}

type ComplexModelC struct {
	Id      uint64 `xorm:"autoincr pk unique" json:"id"`
	Name    string `xorm:"varchar(20)" json:"name"`
	NowTime time.Time
	Count   uint
}

type ComplexModelD struct {
	Id               uint64        `xorm:"autoincr pk unique" json:"id"`
	Name             string        `xorm:"varchar(20)" json:"name"`
	TestString       string        `xorm:"varchar(20)" json:"test_string"`
	TestInt          int           `xorm:"" json:"test_int"`
	TestInt8         int8          `xorm:"" json:"test_int_8"`
	TestInt16        int16         `xorm:"" json:"test_int_16"`
	TestInt32        int32         `xorm:"" json:"test_int_32"`
	TestInt64        int64         `xorm:"" json:"test_int_64"`
	TestUint         uint          `xorm:"" json:"test_uint"`
	TestUint8        uint8         `xorm:"" json:"test_uint_8"`
	TestUint16       uint16        `xorm:"" json:"test_uint_16"`
	TestUint32       uint32        `xorm:"" json:"test_uint_32"`
	TestUint64       uint64        `xorm:"" json:"test_uint_64"`
	TestFloat32      float32       `json:"test_float_32"`
	TestFloat64      float64       `json:"test_float_64"`
	TestTimeDuration time.Duration `json:"test_time_duration"`
	TestTimeTime     time.Time     `json:"test_time_time"`
	TestBool         bool          `json:"test_bool"`
}

type TestUserModel struct {
	Id       uint64 `xorm:"autoincr pk unique" json:"id"`
	UserName string `xorm:"varchar(60) notnull" json:"user_name"`
	Password string `xorm:"varchar(100) notnull" json:"password"`
	Salt     string `xorm:"varchar(40) notnull" json:"salt"`
	Niubi    string `xorm:"varchar(30)"`
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

	engine := database.Engine
	modelList := []interface{}{new(TestModelA), new(TestModelB), new(ComplexModelC), new(ComplexModelD)}

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

	_ = app.Listen(":8080")

}
