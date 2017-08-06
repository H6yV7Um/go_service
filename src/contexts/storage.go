package contexts

import (
	"fmt"
	"libs/mysql"
	"libs/ssdb"
	"libs/gomemcache"
	"log"
	"strings"
	"utils"
	"strconv"
)

// Storage ...
type Storage struct {
	Data map[string][]interface{}
}

type StorageStatsInterface interface {
	Stats() *utils.StorageStats
}

// NewStorage ...
func NewStorage() *Storage {
	return &Storage{Data: make(map[string][]interface{})}
}

func (c *Context) registerStorage() {
	for name, val := range c.Cfg.Storage {
		if _, ok := c.Storage[val.Adapter]; !ok {
			c.Storage[val.Adapter] = NewStorage()
		}
		switch val.Adapter {
		case REDIS:
			for _, uriString := range val.Servers {
				if object, err := utils.NewReidsFromUriString(uriString); err != nil {
					panic("register redis error")
				} else {
					c.Storage[val.Adapter].Data[name] = append(c.Storage[val.Adapter].Data[name], object)
				}
			}
		case SSDB:
			for _, uriString := range val.Servers {
				ret := strings.Split(uriString, ":")
				if len(ret) != 2 {
					log.Printf("invalid ssdb config:%v\n", uriString)
					return
				}
				host := ret[0]
				port := ret[1]
				if object, err := ssdb.NewSsdbPool(host, port); err != nil {
					panic("register ssdb error")
				} else {
					c.Storage[val.Adapter].Data[name] = append(c.Storage[val.Adapter].Data[name], object)
				}
			}
		case MYSQL_NEW:
			for _, uriString := range val.Servers {
				//dbs := mysql.NewPool(uriString, val.Poolsize, val.Hash)
				db, err := mysql.NewMySQL(uriString, val.Hash, val.Poolsize)
				if err == nil {
					c.Storage[val.Adapter].Data[name] = append(c.Storage[val.Adapter].Data[name], db)
				} else {
					log.Fatalf("init db failed, uri=%s err=%v", uriString, err)
				}
			}
		case MEMCACHE_NEW:
			mcClient := memcache.New(val.Servers...)
			if c.Cfg.Switch.EnableMemcacheCompression {
				mcClient.SetFlag(memcache.MEMCACHE_COMPRESSED)
			}
			c.Storage[val.Adapter].Data[name] = append(c.Storage[val.Adapter].Data[name], mcClient)
		}
	}
}

func (c *Context) storeageCount(protocal string, name string) int {
	if storage, ok := c.Storage[protocal]; !ok {
		return -1
	} else {
		if val, ok := storage.Data[name]; !ok {
			return -1
		} else {
			return len(val)
		}
	}
}

func (c *Context) storageGet(protocal string, name string, alias string, index int) (instance interface{}, err error) {
	if storage, ok := c.Storage[protocal]; !ok {
		err = fmt.Errorf("storage get: unknown protocal name %s (forgot to import?)", protocal)
		return nil, err
	} else {
		if val, ok := storage.Data[name]; !ok {
			if alias != "" {
				return c.storageGet(protocal, alias, "", index)
			} else {
				err = fmt.Errorf("storage get: protocal '%s' unknown name %s ", protocal, name)
				return nil, err
			}
		} else {
			if len(val) < index {
				err = fmt.Errorf("storage get: out of index range protocal '%s' unknown name %s index '%d'", protocal, name, index)
				return nil, err
			}

			instance = val[index]
			return instance, nil
		}
	}
}

// Redis ...
func (c *Context) Redis(name string, index ...int) (*utils.Redis, error) {
	findIndex := 0
	if len(index) > 0 {
		findIndex = index[0]
	}

	if instance, err := c.storageGet(REDIS, name, c.redisalias(name), findIndex); err != nil {
		log.Fatalf("Get redis(%v) failed:%v\n", name, err)
		return nil, err
	} else {
		return instance.(*utils.Redis), err
	}
}

// Ssdb ...
func (c *Context) Ssdb(name string, index ...int) (*ssdb.SsdbPool, error) {
	findIndex := 0
	if len(index) > 0 {
		findIndex = index[0]
	}

	if instance, err := c.storageGet(SSDB, name, c.ssdbalias(name), findIndex); err != nil {
		log.Fatalf("Get ssdb(%v) failed:%v\n", name, err)
		return nil, err
	} else {
		return instance.(*ssdb.SsdbPool), err
	}
}

func (c *Context) GetConnInfo() map[string]map[string]int {
	result := make(map[string]map[string]int)
	for protocal, s := range(c.Storage) {
		result[protocal] = make(map[string]int)
		for name, storageList := range(s.Data) {
			if len(storageList) == 0 {
				continue
			}
			storage := storageList[0]
			activeConn := 0
			ifce, implemented := storage.(StorageStatsInterface)
			if implemented {
				activeConn = ifce.Stats().ActiveConn
				result[protocal][name] = activeConn
			} else {
				fmt.Printf("storage stats:unkown storage:%s\n", name)
			}
		}
	}
	return result
}

// GetDBInstance ...
func (c *Context) GetDBInstance(modelName string, alias string, key ...uint64) (ret *mysql.MySQL) {
	ret = mysql.NewEmpty()
	modelInfo, exists := mysql.ModelInfoMap[modelName]
	if !exists {
		utils.WriteLog("error", "modelName not exists:%s", modelName)
		return
	}
	index := 0
	if len(key) > 0 {
		hash := uint64(utils.CRC32(strconv.FormatUint(key[0], 10)))
		index = int(hash % uint64(c.storeageCount(MYSQL_NEW, alias)))
	}
	db, err := c.GetDB(alias, index)
	if err != nil {
		utils.WriteLog("error", "failed to get db %s/%s:%v", modelName, alias, err)
		return
	}
	ret.SetDB(db.GetDB())
	ret.SetModelInfo(&modelInfo)

	return
}

// GetDB ...
func (c *Context) GetDB(name string, index ...int) (*mysql.MySQL, error) {
	findIndex := 0
	if len(index) > 0 {
		findIndex = index[0]
	}

	if instance, err := c.storageGet(MYSQL_NEW, name, c.mysqlalias(name), findIndex); err != nil {
		return nil, err
	} else {
		return instance.(*mysql.MySQL), err
	}
}

func (c *Context) Memcache(name string, index ...int) (*memcache.Client, error) {
	findIndex := 0
	if len(index) > 0 {
		findIndex = index[0]
	}
	
	if instance, err := c.storageGet(MEMCACHE_NEW, name, c.memcachealias(name), findIndex); err != nil {
		log.Println(err.Error())
		return nil, err
	} else {
		return instance.(*memcache.Client), err
	}
}

func (c *Context) release() {
	for adapter, val := range c.Storage {
		for _, conns := range val.Data {
			for _, conn := range conns {
				switch adapter {
				//case MYSQL:
				//	conn.(*utils.DB).Close()
				//case MEMCACHE:
				//	conn.(*utils.Memcache).Close()
				case REDIS:
					conn.(*utils.Redis).Close()
				}
			}
		}
	}
}
