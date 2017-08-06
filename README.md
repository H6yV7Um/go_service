### 依赖的模块：

* github.com/cihub/seelog
* github.com/garyburd/redigo/redis
* gopkg.in/vmihailenco/msgpack.v2
* github.com/go-sql-driver/mysql
* gopkg.in/gcfg.v1


cd /data1/code/service/bin; sudo sh start.sh release-1.0.132
cd /data1/service/8180/bin; sudo sh restart.sh service_8180 release-1.0.133
cd /data1/service/8181/bin; sudo sh restart.sh service_8181 release-1.0.133
