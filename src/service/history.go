package service

/*****
 * feed history module
 *
 * @author guopan1@staff.weibo.com<yiguopan126.com>
 * @date 2015-08-08
 */
import (
	"github.com/garyburd/redigo/redis"
	"strconv"
	"strings"
)

const (
	userFeedHistoryPrefix = "FUSER:"
	userReadHistoryPrefix = "FUSERREAD:"
	imeiFeedHistoryPrefix = "FIMEI:"
	imeiReadHistoryPrefix = "FIMEIREAD:"
)

// Uint64toString ...
func Uint64toString(d uint64) string {
	return strconv.FormatUint(d, 10)
}

func (s *Service) uid2CacheKey(uid uint64) string {
	return userFeedHistoryPrefix + Uint64toString(uid)
}

func (s *Service) aid2CacheKey(aid string) string {
	return imeiFeedHistoryPrefix + aid
}

func (s *Service) usrReadCacheKey(uid uint64) string {
	return userReadHistoryPrefix + Uint64toString(uid)
}

func (s *Service) aidReadCacheKey(aid string) string {
	return imeiReadHistoryPrefix + aid
}

func (s *Service) writeHistory(cacheKey string, redisHashKey uint64, articleIDs *[]string) {
	if cacheKey == "" || articleIDs == nil {
		return
	}

	redisClient, err := s.Ctx.RedisClient2Hash("user_w", redisHashKey)
	if err != nil {
		return
	}

	for _, articleID := range *articleIDs {
		redisClient.Do("LPUSH", cacheKey, articleID)
	}

	redisClient.Do("LTRIM", cacheKey, 0, 2000)
	redisClient.Do("EXPIRE", cacheKey, 86400*5)
	redisClient.Do("EXEC")
}

func (s *Service) readHistory(cacheKey string, redisHashKey uint64) map[string]uint64 {
	m := make(map[string]uint64)
	redisClient, err := s.Ctx.RedisClient2Hash("user_r", redisHashKey)
	if err != nil {
		return nil
	}

	if history, err := redis.Strings(redisClient.Do("LRANGE", cacheKey, 0, 2000)); err == nil {
		for _, v := range history {
			elem := strings.Split(v, "~")
			if len(elem) >= 2 {
				i, err := strconv.ParseUint(elem[1], 10, 64)
				if err == nil {
					m[elem[0]] = i
				} else {
					m[elem[0]] = 1
				}
			} else {
				m[elem[0]] = 1
			}
		}
	}

	return m
}
