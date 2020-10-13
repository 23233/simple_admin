## [英文](https://github.com/23233/simple_admin/blob/master/README.md)

## 关于simple_admin

非常感觉iris框架极其开发者的帮助~

现在扯犊子,这个库主要解决的一个小问题:

#### xorm 或者 gorm 没有一个简单的结构模型管理后台

这我相信90%的golang开发者在使用web框架的时候都是基于struct生成的各类表模型

但是这两个库都没有 就很难受 想想python django自带管理后台 再看看php 泪牛满面

我去提了issue,并且等待很长时间都没有这种计划的支持 可能也许是没有泛型 写这种玩意没有动力吧

所以没法子 自己业务又需要 就简单地搞了一个 

但是时间和精力都有限 只能从我自己的业务出发 所以我使用的技术栈:

* iris
* xorm
* casbin -> rbac
* react -> ant.design

希望可以帮到相同技术栈的朋友 后期若有必要会进行基础适配 比如说支持gin 支持gorm

## 截图预览
![welcome](https://raw.githubusercontent.com/23233/simple_admin/master/_preview/welcome.png)
![dashBoard](https://raw.githubusercontent.com/23233/simple_admin/master/_preview/dashBoard.png)
![dataList](https://raw.githubusercontent.com/23233/simple_admin/master/_preview/dataList.png)
![guest](https://raw.githubusercontent.com/23233/simple_admin/master/_preview/guest.png)
![userManage](https://raw.githubusercontent.com/23233/simple_admin/master/_preview/userManage.png)

## 使用方法

安装
```
go get https://github.com/23233/simple_admin
```
___
案例 ->  [https://github.com/23233/simple_admin/tree/master/_examples](https://github.com/23233/simple_admin/tree/master/_examples)

___

预备 -> 定义你的表数据struct
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
预备 -> 初始化你的xorm引擎

```

Engine, err = xorm.NewEngine("mysql", dbUrl)

```

预备 -> 初始化iris
```
app := iris.New()
```

起飞 -> 注册simple_admin
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

起飞 -> 打开浏览器访问 http://127.0.0.1:8080/admin
```
app.Listen(":8080")
```

## 注意事项
* ~~最重要的 不支持 xorm  `deleted` 标签~~ *删除数据都是使用 Unscoped 方法 也就是彻底物理删除*
* 最好使用自带的后台用户管理模型 减少麻烦

## 支持模型自定义tag
```golang
example: sp("key:value") sp("key(value)") sp("key('value')")
```
* autogen  -> 标记这个字段是代码生成不经过用户
```golang
sp("autogen")
```
* lineTo -> 如果使用自定义操作 这个tag对应的字段会在新增时自动填充选择的值为默认值
```golang
sp("lineTo(Id)")
```
* fk -> 外键支持,1对1 1多多,目前只支持主键id 必须拥有id字段* 
```golang
sp("fk('ComplexModelC')") 
sp("fk('ComplexModelC') multiple")
```
* tag -> 前端显示的标签 目前支持: img 
```golang
sp:"tag(img)"
```

## 事件支持
* SpInsertBefore
* SpInsertAfter
* SpUpdateBefore
* SpUpdateAfter
* SpDeleteBefore(id uint64)  // 因为删除使用 unscoped 物理删除 所以仅接收一个id字段 
* SpDeleteAfter(id uint64)  // 因为删除使用 unscoped 物理删除 所以仅接收一个id字段 

## 自定义操作
* 方法名称只要是 SpAction开头就行 比如说 SpAction() SpAction123() 都行 必须有返回 simple_admin.CustomAction 

## 更多特性支撑
- [] full test (不打算近期支持)
- [] gorm support (看反馈)
- [] gin support (看反馈)
- [x] dashboard
- [x] simple event monitor
- [x] add spider visit monitor options enable!
- [x] custom action 
- [] fine permission manage (看反馈)
- [] support micro frontend , use [qiankun](https://github.com/umijs/qiankun) (这样大家就可以自定义页面 二次开发)
- [] beat more features  
