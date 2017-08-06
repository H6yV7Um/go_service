package tools

import (
	"net/http"
	"fmt"
	"contexts"
	"strings"
	"encoding/base64"
)

type Session struct {
	username 	string
	auth 		bool
}

const (
	sessionKeyPrefix = "_SESSION:"
	cookieName = "TTRC"
)

func CheckLoginCookie(r *http.Request) (bool, *Session) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		fmt.Printf("get cookie failed:%v\n", err)
		return false, nil
	}
	sid := c.Value
	fmt.Printf("sid=%v\n", sid)
	ctx := contexts.GetContext()
	session, found := ctx.Cache.Get(sessionKeyPrefix + sid)
	if !found {
		fmt.Printf("session not found for sid:%v\n", sid)
		return false, nil
	}
	s := session.(*Session)
	return s.auth, s
}

func Unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Toutiao User Login"`)
	w.WriteHeader(http.StatusUnauthorized)
}

func CheckLoginBasic(w http.ResponseWriter, r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		Unauthorized(w)
		return false
	}

	auths := strings.SplitN(auth, " ", 2)
	if len(auths) != 2 {
		Unauthorized(w)
		return false
	}
	authMethod := auths[0]
	authB64 := auths[1]
	switch authMethod {
	case "Basic":
		authstr, err := base64.StdEncoding.DecodeString(authB64)
		if err != nil {
			fmt.Println(err)
			Unauthorized(w)
			return false
		}
		userPwd := strings.SplitN(string(authstr), ":", 2)
		if len(userPwd) != 2 {
			fmt.Println("error")
			Unauthorized(w)
			return false
		}
		username := userPwd[0]
		password := userPwd[1]
		if username == "admin" && password == "ur" {
			return true
		} else {
			fmt.Println(err)
			Unauthorized(w)
			return false
		}
	default:
		Unauthorized(w)
		return false
	}
	return false
}
