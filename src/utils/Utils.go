package utils

import (
	"hash/crc32"
	"strconv"
	"strings"
	"unicode"
	"net"
	"errors"
	"runtime"
)

type StorageStats struct {
	ActiveConn	int
}

// String ...
func String(reply interface{}) string {
	switch reply := reply.(type) {
	case []byte:
		return string(reply)
	case string:
		return reply
	case nil:
		return ""
	}
	return ""
}

// Atoi ...
func Atoi(str string, defaultVal ...int) int {
	if val, err := strconv.Atoi(str); err == nil {
		return val
	} else if len(defaultVal) > 0 {
		return defaultVal[0]
	} else {
		return 0
	}
}

// Hash ...
func Hash(key string) uint64 {
	return uint64(CRC32(key))
}

// CRC32 ...
func CRC32(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

// KeyExists ...
func KeyExists(m map[string]interface{}, k string) bool {
	if m == nil {
		return false
	}
	_, exists := m[k]
	return exists
}

// KeysExists ...
func KeysExists(m map[string]interface{}, keys []string) bool {
	if m == nil {
		return false
	}
	for _, k := range keys {
		if !KeyExists(m, k) {
			return false
		}
	}
	return true
}

// MapGetKey ...
func MapGetKey(m map[string]interface{}, key string, defaultValue string) interface{} {
	if m == nil {
		return defaultValue
	}
	if v, exists := m[key]; exists {
		return v
	}
	return defaultValue
}

// GetValueByPath ...
func GetValueByPath(src map[string]interface{}, fields string) (result interface{}) {
	o := src
	path, _ := getTypeAndPath(fields)
	for i:=0; i<len(path); i++ {
		o2, exists := o[path[i]]
		if !exists {
			//fmt.Printf("not found %s\n", path[i])
			result = ""
			return
		}
		if i == len(path)-1 {
			/*switch t {
			case "int":
				fmt.Printf("t=int f=%s\n", path[i])
				result, _ = strconv.Atoi(o2.(string))
			case "float64":
				fmt.Printf("t=float64 f=%s\n", path[i])
				result, _ = strconv.Atoi(o2.(string))
			case "bool":
				result = o2.(string) == "true"
			default:
				result = o2
			}*/
			//fmt.Printf("k=%s value=%v\n", path[i], o2)
			result = o2
		} else {
			var ok bool
			o, ok = o2.(map[string]interface{})
			if !ok {
				result = ""
				return
			}
		}
	}
	//fmt.Printf("f=%s k=%s value=%v\n", fields, path, result)
	return
}

// GetStringValueByPath ...
func GetStringValueByPath(src map[string]interface{}, fields string) string {
	v := GetValueByPath(src, fields)
	if v == nil {
		return ""
	}
	return v.(string)
}

// GetBoolValueByPath ...
func GetBoolValueByPath(src map[string]interface{}, fields string) bool {
	v := GetValueByPath(src, fields)
	if v == nil {
		return false
	}
	return v.(bool)
}

// GetIntValueByPath ...
func GetIntValueByPath(src map[string]interface{}, fields string) uint64 {
	v := GetValueByPath(src, fields)
	if v == nil {
		return 0
	}
	switch v := v.(type) {
	case float64:
		return uint64(v)
	case uint64:
		return v
	case int:
		return uint64(v)
	case int64:
		return uint64(v)
	}
	WriteLog("info", "GetIntValueByPath failed, invalid json value for %s:%v\n", fields, v)
	return 0
}

// GetFloatValueByPath ...
func GetFloatValueByPath(src map[string]interface{}, fields string) float64 {
	v := GetValueByPath(src, fields)
	if v == nil {
		return 0
	}
	switch v := v.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	}
	WriteLog("error", "GetFloatValueByPath invalid json value for %s:%v\n", fields, v)
	return 0
}

func getTypeAndPath(field string) (path []string, fieldType string) {
	kv := strings.Split(field, ":")
	if len(kv) != 2 {
		fieldType = "string"
		path = strings.Split(kv[0], "/")
	} else {
		fieldType = kv[0]
		path = strings.Split(kv[1], "/")
	}
	return
}

/*func SafeCopyAttributes(src map[string]interface{}, dest map[string]interface{}, fields map[string]string) {
	for k, _ := range(fields) {
		value := getValueByPath(src, k)
		fmt.Printf("k=%s value=%v\n", k, value)
	}
}*/

// CheckBadArg ...
func CheckBadArg(arg string, maxLength int) bool {
	words := []string {"select", "sleep"}
	/*if len(arg) > maxLength {
		return false
	}*/
	s := strings.ToLower(arg)
	for _, word := range(words) {
		if strings.Contains(s, word) {
			return false
		}
	}
	return true
}

// CheckOid ...
func CheckOid(oid string) bool {
	if oid == "" {
		return false
	}
	s := strings.ToLower(oid)
	if !strings.Contains(s, ":") && !strings.HasPrefix(s, "zt_") {
		return false
	}
	if !CheckBadArg(s, 100) {
		return false
	}
	return true
}

// CheckImei ...
func CheckImei(oid string) bool {
	s := strings.ToLower(oid)
	if !CheckBadArg(s, 40) {
		return false
	}
	return true
}

// IsNewID ...
func IsNewID(id string) bool {
	if strings.Contains(id, ":") {
		return false
	}
	for _, d := range(id) {
		if !unicode.IsDigit(d) {
			return false
		}
	}
	return true
}

// GetInternalIpAddr ...
func GetInternalIpAddr() (string, error) {
	_, ipnet1, _ := net.ParseCIDR("10.0.0.0/8")
	_, ipnet2, _ := net.ParseCIDR("172.16.0.0/12")
	_, ipnet3, _ := net.ParseCIDR("192.168.0.0/16")
	privateIpNet := []*net.IPNet{ipnet1, ipnet2, ipnet3}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		WriteLog("error", "getInternalIpAddr failed:" + err.Error())
		return "", err
	}
	internalIpAddrs := []string{}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			ip := ipnet.IP.String()
			WriteLog("info", "- query interface, ip=%s", ip)
			if ipnet.IP.To4() == nil {
				WriteLog("info", "    not ipv4")
				continue
			}
			for _, i := range(privateIpNet) {
				if i.Contains(ipnet.IP) {
					WriteLog("info", "    is internal")
					internalIpAddrs = append(internalIpAddrs, ipnet.IP.String())
				}
			}
		}
	}

	// 如果有多个内网ip，返回第二个
	switch {
	case len(internalIpAddrs) == 1:
		return internalIpAddrs[0], nil
	case len(internalIpAddrs) > 1:
		return internalIpAddrs[1], nil
	default:
		return "", errors.New("internal ip not found")
	}
}

func Min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func IsCardArticle(idStr string) bool {
	return CheckBid(idStr, ARTICLE_BID_CARD)
}

func IsTopArticle(idStr string) bool {
	return CheckBid(idStr, ARTICLE_BID_TOPARTICLE)
}

func CheckBid(idStr string, bid int64) bool {
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		return false
	}
	idGen, err := Decode(id)
	if err != nil {
		return false
	}
	if idGen.Bid == bid {
		return true
	}
	return false
}

func InArray(arr []string, s string) bool {
	for _, a := range(arr) {
		if a == s {
			return true
		}
	}
	return false
}

func IsSpecial(idStr string) bool {
	if strings.HasPrefix(idStr, "zt_") {
		return true
	} else {
		return false
	}
}

var m runtime.MemStats
func DebugHeapObjects (msg string) {
	runtime.ReadMemStats(&m)
	WriteLog("debug", "%s heap_objects=%d", msg, m.HeapObjects)
}