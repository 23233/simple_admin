package simple_admin

import (
	"fmt"
	"github.com/pkg/errors"
	"math/rand"
)

func RandStringBytes(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func StringsContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false

}

func GetMapValues(a map[string]string) []string {
	d := make([]string, len(a))
	for _, v := range a {
		d = append(d, v)
	}
	return d
}

func MsgLog(msg string) error {
	return errors.New(fmt.Sprintf("%s", msg))
}
