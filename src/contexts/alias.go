package contexts

import (
	"strings"
)

// Alias ...
type Alias struct {
	Data map[string]string
}

// NewAlias ...
func NewAlias() *Alias {
	return &Alias{Data: make(map[string]string)}
}

func (c *Context) registerAlias() {
	for adapter, val := range c.Cfg.Alias {
		if _, ok := c.Alias[adapter]; !ok {
			c.Alias[adapter] = NewAlias()
		}

		for _, line := range val.Values {
			elem := strings.Split(line, ":")
			if len(elem) == 2 {
				c.Alias[adapter].Data[elem[0]] = elem[1]
			}
		}
	}
}

func (c *Context) alias2RealName(adapter string, alias string) string {
	if val, ok := c.Alias[adapter]; !ok {
		return ""
	} else {
		return val.Data[alias]
	}
}

func (c *Context) redisalias(alias string) string {
	return c.alias2RealName(REDIS, alias)
}

func (c *Context) ssdbalias(alias string) string {
	return c.alias2RealName(SSDB, alias)
}

func (c *Context) memcachealias(alias string) string {
	return c.alias2RealName(REDIS, alias)
}

func (c *Context) mysqlalias(alias string) string {
	return c.alias2RealName(REDIS, alias)
}
