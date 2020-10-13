## [中文](https://github.com/23233/simple_admin/blob/master/README_ZH.md)

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

## preview

## use

Install
```
go get https://github.com/23233/simple_admin
```
___
Examples ->  [https://github.com/23233/simple_admin/tree/master/_examples](https://github.com/23233/simple_admin/tree/master/_examples)

___

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
    Prefix:    "/admin", // path prefix like app.Prefix("/admin")
})
```

Go -> Run you app  open browser http://127.0.0.1:8080/admin Yes god job
```
app.Listen(":8080")
```

## warning
* ~~first, now don't support xorm  `deleted` tag~~ *delete use Unscoped methods!*
* the best you do not use custom usermodel , admin isolation is good!

## model custom tags (sp) now support
```golang
example: sp("key:value") sp("key(value)") sp("key('value')")
```
* autogen  -> mark columns is code auto generate not handle
```golang
sp("autogen")
```
* lineTo -> if custom action , default value use this define for select row
```golang
sp("lineTo(Id)")
```
* fk -> foreign key , support one to tone and many to one , if use multiple is many to one , *just only support id columns , must be have id field !!!* 
```golang
sp("fk('ComplexModelC')") 
sp("fk('ComplexModelC') multiple")
```
* tag -> front show whats ? now support custom tag , default show text , options: img  
```golang
sp:"tag(img)"
```

## event support 
* SpInsertBefore
* SpInsertAfter
* SpUpdateBefore
* SpUpdateAfter
* SpDeleteBefore(id uint64)  // because delete use unscoped methods so this func receive a uint64 type params
* SpDeleteAfter(id uint64)  // because delete use unscoped methods so this func receive a uint64 type params

## custom action
* func name must use SpAction , just like SpAction() SpAction123() is ok , must return simple_admin.CustomAction struct !

## todo features
- [] full test
- [] gorm support
- [x] dashboard
- [x] simple event monitor
- [x] add spider visit monitor options enable!
- [x] custom action 
- [] fine permission manage
- [] beat more features
