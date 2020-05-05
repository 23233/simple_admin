package database

import (
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
