package simple_admin

import (
	"github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris/v12"
	"time"
)

var MySecret = []byte("8657684ae02840ead423e0d781a7a0c5")

// 自定义JWT
// 使用办法 中间层 handler.CustomJwt.Serve, handler.TokenToUserUidMiddleware,user handler
var CustomJwt = jwt.New(jwt.Config{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return MySecret, nil
	},
	Expiration:          true,
	CredentialsOptional: true,
	SigningMethod:       jwt.SigningMethodHS256,
})

// 登录token存储信息 记录到上下文中
func TokenToUserUidMiddleware(ctx iris.Context) {
	user := ctx.Values().Get("jwt").(*jwt.Token)
	jwtData := user.Claims.(jwt.MapClaims)
	userUid := jwtData["userUid"].(string)
	// 这里可以遍历所有的token信息
	//for key, value := range jwtData {
	//	_, _ = ctx.Writef("%s = %s", key, value)
	//}

	ctx.Values().Set("uid", userUid)
	ctx.Next() // execute the next handler, in this case the main one.
}

// 生成token
func GenJwtToken(userUid string) string {
	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userUid": userUid,
		"exp":     time.Now().Add(time.Hour * 120).Unix(), //过期时间 120小时
	})
	tokenString, _ := token.SignedString(MySecret)
	return tokenString
}
