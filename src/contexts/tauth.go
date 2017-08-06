package contexts

import(
	"fmt"
	"strings"
	"strconv"
	"crypto/sha1"
	"encoding/base64"
	"crypto/hmac"
	"net/url"
	"utils"
)

var (
	tauthToken = ""
	tauthTokenSecret = ""
)

const (
	tauthUID = 5334460004
)

func (c *Context) getToken() {
	r, err := c.Redis("subscribed_r")
	if err != nil {
		utils.WriteLog("error", "get subscribed_r redis failed")
		return
	}
	tauthKey := fmt.Sprintf("tauth2_token:%s:10,172", APPKEY)
	tauthValue := r.Get(tauthKey)
	if tauthValue == nil {
		utils.WriteLog("error", "get token from redis failed, key=%s", tauthKey)
		return
	}

	tokenSlice := strings.Split(string(tauthValue.([]byte)), "\t")
	tauthToken = tokenSlice[0]
	tauthTokenSecret = tokenSlice[1]
}

// TAuthApiRequest ...
func (c *Context) TAuthApiRequest() string {
	if tauthToken == "" || tauthTokenSecret == "" {
		c.getToken()
		if tauthToken == "" || tauthTokenSecret == "" {
			utils.WriteLog("error", "get tauth token failed")
			return ""
		}
	}
	param := "uid=" + strconv.FormatInt(tauthUID, 10)
	h := hmac.New(sha1.New, []byte(tauthTokenSecret))
	_, err := h.Write([]byte(param))
	if err != nil {
		utils.WriteLog("error", "hmac generate authorization error" + err.Error())
		return ""
	}
	s := string(h.Sum(nil))
	tauthSignature := base64.StdEncoding.EncodeToString([]byte(s))
	authorization := `TAuth2 token="` + url.QueryEscape(tauthToken) + `",sign="` + url.QueryEscape(tauthSignature) + `",param="` + url.QueryEscape(param) + `"`
	return authorization
}
