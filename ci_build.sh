#!/usr/bin/env bash

export GOPATH=/data1/jenkins/jobs/toutiao_service/workspace:/data1/go

echo "release=${release}"
if [ ${release} = "true" ]; then
    # git 必须是2.1以上的版本才支持--sort选项
    tag=`/usr/local/git/bin/git tag -l "release*" --sort version:refname|tail -1`
    if [ -z "$tag" ]; then
        tag="release-1.0.1"
    fi
    echo "build release, tag=$tag"
else
    tag=`git tag -l "dev*"|tail -1`
    if [ -z "$tag" ]; then
        tag="dev-1.0.1"
    fi
    echo "build dev, tag=$tag"
fi
fields=(${tag//./ })
fields_number=${#fields[@]}
if [ $fields_number -ne "3" ]; then
    echo "invalid tag:$tag fields_number=$fields_number"
    exit 1
fi
((build_number=${fields[2]}+1))
build_tag=${fields[0]}.${fields[1]}.$build_number
echo "build_tag=$build_tag"
>version.txt
echo $build_tag > version.txt
>build_name.txt
echo ""${build_tag}_`date +%Y-%m-%d`"" > build_name.txt
go build -o service-$build_tag -ldflags "-X service.buildVersion=$build_tag -X service.buildDate=`date '+%Y-%m-%d_%H:%M:%S'` -X service.buildGithash=`git rev-parse HEAD`" src/main/main.go
ret=$?
if [ $ret -ne "0" ]; then
    echo "build failed, result=$ret"
    exit $ret
fi

go build -o rpc_client src/client/main.go

if [ ${release} = "true" ]; then
    echo "add tag $build_tag to git "
    echo "https://weibo_platform_ci:123qweASD@gitlab.weibo.cn" > /tmp/git4009600337861275507.credentials
    git config --local credential.username weibo_platform_ci
    git config --local credential.helper store --file=/tmp/git4009600337861275507.credentials
    git tag -a $build_tag -m "build version $build_tag"
    git push origin --tags
    git config --local --remove-section credential
    rm -f /tmp/git4009600337861275507.credentials

    sh deploy.sh $build_tag
    echo $build_tag > /data1/jenkins/email-templates/build_tag.txt
else
    killall service-tools
    cp rpc_client /data1/service/static/downloads
    cp service-$build_tag /data1/service/service-tools
    cd /data1/service
    daemonize -E BUILD_ID=dontKillMe -c /data1/service /data1/service/service-tools -confdir conf/tools
fi

