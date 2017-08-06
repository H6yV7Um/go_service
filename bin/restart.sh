# sh restart.sh service_8180 release-1.0.100
supervisorctl stop $1
cp -f service-$2 service
supervisorctl start $1
