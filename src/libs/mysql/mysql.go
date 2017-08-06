package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"strings"
	"sync"
	"time"
	. "utils"
	"log"
	"reflect"
)

const (
	DefaultMaxConn     = 128
	DefaultMaxIdleConn = 32
)

// Db ...
type Db struct {
	m *sync.Mutex
	r map[string]*MySQL
}

type Field []string // {"id", "oid", "title"}
type Where []string // {"time>12345", "category=1"}
type Order []string // {"name1", "desc"}
type Limit []string // {offset, count}
type Rows map[int]map[string]interface{}

// MySQL ...
type MySQL struct {
	db          *sql.DB
	hash        string
	modelInfo	*ModelInfo
}

func NewMySQL(dsn string, hash string, poolSize string) (*MySQL, error) {
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	maxConn := DefaultMaxConn
	maxIdleConn := DefaultMaxIdleConn
	poolSizeConfig := strings.Split(poolSize, ",")
	if len(poolSizeConfig) == 2 {
		n1, err1 := strconv.Atoi(poolSizeConfig[0])
		n2, err2 := strconv.Atoi(poolSizeConfig[1])
		if err1 == nil && err2 == nil {
			maxConn = n1
			maxIdleConn = n2
		}
	}
	conn.SetMaxOpenConns(maxConn)
	conn.SetMaxIdleConns(maxIdleConn)
	log.Printf("init mysql, dsn=%s, hash=%s, maxOpenConns=%d, maxIdleConns=%d\n",
		dsn, hash, maxConn, maxIdleConn)
	return &MySQL{
		db:     conn,
		hash: 	hash,
	}, nil
}

func (this *MySQL)Stats() *StorageStats {
	return &StorageStats {
		ActiveConn:this.db.Stats().OpenConnections,
	}
}

func (m *MySQL) Release() {
}

func (m *MySQL) SetMaxOpenConns(n int) {
	m.db.SetMaxOpenConns(n)
}

func (m *MySQL) SetMaxIdleConns(n int) {
	m.db.SetMaxIdleConns(n)
}

func (m *MySQL) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.db.Exec(query, args)
}

type ModelInfo struct {
	modelName 			string
	modelIfce 			interface{}
	fieldNum			int
	fieldsNames 		[]string
	fieldsTypes			[]reflect.Kind
	fieldsNamesIndex 	map[string]int
}

var ModelInfoMap map[string]ModelInfo

func init() {
	ModelInfoMap = make(map[string]ModelInfo)
}

func RegisterModel(m interface{}) {
	modelValue := reflect.ValueOf(m)
	modelElem := modelValue.Elem()
	modelName := strings.ToLower(modelElem.Type().Name())
	fieldNum := modelElem.NumField()

	var modelInfo ModelInfo
	modelInfo.modelName = modelName
	modelInfo.modelIfce = m
	modelInfo.fieldNum = fieldNum
	modelInfo.fieldsNames = make([]string, fieldNum)
	modelInfo.fieldsTypes = make([]reflect.Kind, fieldNum)
	modelInfo.fieldsNamesIndex = make(map[string]int)

	for i := 0; i < modelElem.NumField(); i++ {
		f := modelElem.Field(i)
		f_name := strings.ToLower(modelElem.Type().Field(i).Name)
		modelInfo.fieldsNames[i] = f_name
		modelInfo.fieldsNamesIndex[f_name] = i
		modelInfo.fieldsTypes[i] = f.Type().Kind()
	}
	ModelInfoMap[modelName] = modelInfo
}

func NewEmpty() *MySQL {
	return &MySQL{db:nil}
}

func (m *MySQL) GetDB() *sql.DB {
	return m.db
}

func (m *MySQL) SetDB(db *sql.DB)  {
	m.db = db
}

func (m *MySQL) SetModelInfo(info *ModelInfo)  {
	m.modelInfo = info
}

func (m *MySQL) ParseRows(rs *sql.Rows) ([]interface{}, error) {
	columns, err := rs.Columns()
	if err != nil {
		WriteLog("error", "get table columns failed:%v", err)
		return nil, err
	}

	scanArgs := make([]interface{}, len(columns))
	fieldsInfo := make([]interface{}, m.modelInfo.fieldNum)
	for i := range scanArgs {
		columnName := columns[i]
		fieldIndex, exists := m.modelInfo.fieldsNamesIndex[columnName]
		if !exists {
			fmt.Printf(">>>> field %s not found!\n", columnName)
			return nil, errors.New("field not found")
		}
		scanArgs[i] = &(fieldsInfo[fieldIndex])
	}

	result := []interface{}{}
	for rs.Next() {
		err := rs.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		o := reflect.New(reflect.TypeOf(m.modelInfo.modelIfce).Elem())
		for i, value := range scanArgs {
			columnName := columns[i]
			fieldIndex, _ := m.modelInfo.fieldsNamesIndex[columnName]
			f := o.Elem().FieldByName(strings.Title(columnName))
			v := ""
			if value != nil {
				v2 := value.(*interface{})
				if v2 != nil && *v2 != nil {
					v = string((*v2).([]byte))
				} else {
					v = ""
				}
			}
			switch m.modelInfo.fieldsTypes[fieldIndex] {
			case reflect.Int:
				i64, _ := strconv.ParseInt(v, 10, 0)
				f.Set(reflect.ValueOf(int(i64)))
			case reflect.Int64:
				i64, _ := strconv.ParseInt(v, 10, 0)
				f.Set(reflect.ValueOf(i64))
			case reflect.String:
				f.Set(reflect.ValueOf(v))
			default:
				fmt.Printf("unknown field type %v for %s v=%v\n", m.modelInfo.fieldsTypes[fieldIndex], columnName, v)
				f.Set(reflect.ValueOf(v))
			}
		}
		result = append(result, o.Interface())
	}
	return result, nil
}

func (m *MySQL) Query(query string) (result []interface{}, err error) {
	if m.db == nil {
		fmt.Printf("empty db instance")
		err = errors.New("empty db instance")
		return
	}

	words := []string {"sleep"}
	s := strings.ToLower(query)
	for _, word := range(words) {
		if strings.Contains(s, word) {
			err = errors.New("invalid sql:" + query)
			return
		}
	}

	now := time.Now().UnixNano()
	rows, err := m.db.Query(query)
	if err == nil {
		result, err = m.ParseRows(rows)
	}
	WriteLog("debug", "sql=%v time_used=%d", query, (time.Now().UnixNano()-now)/1000000)
	m.Release()

	return
}

func MysqlRealEscapeString(value string) string {
	replace := map[string]string{"'":`\'`, "\\0":"\\\\0", "\n":"\\n", "\r":"\\r", `"`:`\"`, "\x1a":"\\Z"}

	value = strings.Replace(value, "\\", "\\\\", -1)
	for b, a := range replace {
		value = strings.Replace(value, b, a, -1)
	}

	return value;
}

func (m *MySQL) Update(tableName string, params map[string]interface{}) (int64, error) {
	var fields []string
	var values = ""
	var updateStatement = ""
	for k, v := range params {
		if k == "" || v == nil {
			continue
		}
		fields = append(fields, fmt.Sprintf("`%s`", k))
		if updateStatement == "" {
			updateStatement = fmt.Sprintf("`%s`=", k)
		} else {
			updateStatement += fmt.Sprintf(", `%s`=", k)
		}
		switch v := v.(type) {
		case int, int64, uint, uint64, float64:
			if values == "" {
				values = fmt.Sprintf("%d", v)
			} else {
				values += fmt.Sprintf(", %d", v)
			}
			updateStatement += fmt.Sprintf("%d", v)
		case string:
			vs := MysqlRealEscapeString(v)
			if values == "" {
				values = fmt.Sprintf("\"%s\"", vs)
			} else {
				values += fmt.Sprintf(", \"%s\"", vs)
			}
			updateStatement += fmt.Sprintf("\"%s\"", vs)
		default:
			return 0, errors.New(fmt.Sprintf("invalid values:%v", v))
		}
	}
	query := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s) ON DUPLICATE KEY UPDATE %s",
		tableName, strings.Join(fields, ", "), values, updateStatement)

	now := time.Now().UnixNano()
	_, err := m.db.Query(query)
	WriteLog("debug", "sql=%v time_used=%d err=%v\n", query, (time.Now().UnixNano()-now)/1000000, err)
	if err != nil {
		return 0, err
	}
	return 1, nil
}
