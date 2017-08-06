#!/usr/bin/env bash

if [ $# -lt 1 ]
then
    echo "version not given"
    exit
fi

go build -o service-$1 -ldflags "-X service.buildVersion=$1 -X service.buildDate=`date '+%Y-%m-%d_%H:%M:%S'` -X service.buildGithash=`git rev-parse HEAD`" src/main/main.go

#rsync --progress service-$1 s1::toutiao_service/bin/service-$1
#rsync --progress service-$1 s2::toutiao_service/bin/service-$1
#rsync --progress service-$1 s3::toutiao_service/bin/service-$1
