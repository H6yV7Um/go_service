import redis
import time

def get_all_push_users():
    key = "push_user_list"
    ret_list = []
    start = 0
    step = 3000
    end = start + step
    conf = {'host': 'rs7019.hebe.grid.sina.com.cn', 'port': 7019}
    r = redis.Redis(**conf)
    total = r.llen(key)
    i = 0
    run = True
    while run:
        try:
            ret = r.lrange(key, start, end)
            if not ret:
                print "ret", ret, start, end
                break
            for line in ret:
                if i % 10000 == 0:
                    print "%d of %d processed, %d users" % (i, total, len(ret_list))
                    if i > 20000:
                        run = False
                        break
                i += 1
                items = line.split('`')
                if len(items) != 9:
                    continue
                uid = 0
                imei = ""
                try:
                    uid, imei = int(items[2]), items[4]
                except Exception, e:
                    uid = 0
                    imei = items[4]
                if uid == 0 and imei == "":
                    continue
                ret_list.append((uid, imei))
        except Exception, error:
            print("redis_push lrange( %s, %s, %s) error:%s" % (key, start, end, error))
            time.sleep(5)
            continue
        start += step + 1
        end += step + 1
    print("total=%d got=%d\n" % (total, len(ret_list)))
    return ret_list

users = get_all_push_users()

conf2 = {'host': 'rs7123.mars.grid.sina.com.cn', 'port': 7123}
r2 = redis.Redis(**conf2)
for (uid, imei) in users:
    if uid == 0:
        s_key = "imei_" + imei
    else:
        s_key = "user_" + str(uid)
    ret = r2.hget("subscribed", s_key)
    print s_key, ret
    #exists = r2.hexists("subscribed", s_key)
    #if exists:
    #    print "OK ", uid, imei
    #else:
    #    print "BAD ", uid, imei
