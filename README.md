## about simple_admin

First time , very thanks for iris framework and developer to helps~

Now talk something , this library just resolve a small and simple problem :

#### Xorm or gorm Why not provide a simple or small struct model visible dashboard

My trust 99% golang uses people need this 

Wait long time not has , start with me , but i time and ability is small and limited !

So I'm just use i familiar technology :

* iris
* xorm
* casbin -> rbac
* react -> ant.design

Hope to help everyone !

## use

Install
```cassandraql
go get https://github.com/23233/simple_admin
```

Ready -> Defined you struct
```
type TestModelA struct {
	Id   uint64 `xorm:"autoincr pk unique" json:"id"`
	Name string `xorm:"varchar(20)"`
}

type TestModelB struct {
	Id   uint64 `xorm:"autoincr pk unique" json:"id"`
	Desc string `xorm:"varchar(60)"`
}
```
Ready -> Defined you xorm engine

```

Engine, err = xorm.NewEngine("mysql", dbUrl)

```

Ready -> Defined iris application
```
app := iris.New()
```

Go -> Register simple_admin
```
modelList := []interface{}{new(TestModelA), new(TestModelB)}

_, err := simple_admin.New(simple_admin.Config{
    Engine:    engine,
    App:       app,
    ModelList: modelList,
    Name:      "app name",
    RunSync:   true, // this is xorm sync2
    EnableReg: false, // if false web front not use reg 
    Prefix:    "/admin", // path prefix like app.Prefix("/admin")
})
```

Go -> Run you app  open browser http://127.0.0.1:8080/admin Yes god job
```
app.Listen(":8080")
```


## todo features
- [] gorm support
- [] custom action
- [] fine permission manage
- [] beat more features