#!/usr/bin/sh

if [ $# -lt 1 ]
then
    echo "version not given"
    exit
fi

#for i in "172.16.181.231" "172.16.181.138" "172.16.181.234" "10.73.13.161" "10.73.13.187"; do
#    echo "sync file to $i..."
#    rsync --progress service-$1 $i::toutiao_service_old/bin/service-$1
#    rsync --progress config.ini $i::toutiao_service_old/config.ini.new
#
#    rsync --progress service-$1 $i::toutiao_service/8180/bin/service-$1
#    rsync --progress config_8180.ini $i::toutiao_service/8180/config_8180.ini.new
#
#    rsync --progress service-$1 $i::toutiao_service/8181/bin/service-$1
#    rsync --progress config_8181.ini $i::toutiao_service/8181/config_8181.ini.new
#done

for i in "172.16.181.231" "172.16.181.138" "172.16.181.234" "10.73.13.161" "10.73.13.187"; do
    echo "sync file to $i..."
    rsync --progress service-$1 $i::toutiao_service_old/bin/service-$1
    rsync --progress config.ini $i::toutiao_service_old/config.ini.new

    rsync --progress service-$1 $i::toutiao_service/8180/bin/service-$1
    rsync --progress -r conf/core/* $i::toutiao_service/8180/conf/core

    rsync --progress service-$1 $i::toutiao_service/8181/bin/service-$1
    rsync --progress -r conf/service/* $i::toutiao_service/8181/conf/service
done