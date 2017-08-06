#! /usr/bin/bash

rm ./lint_message
golint ./src/service/* | grep -v "struct field Uid" | grep -v "struct field Id" > ./lint_message
golint ./src/main/* | grep -v "struct field Uid" | grep -v "struct field Id" >> ./lint_message
golint ./src/tools/* | grep -v "struct field Uid" | grep -v "struct field Id" >> ./lint_message
#golint ./src/utils/* | grep -v "struct field Uid" | grep -v "struct field Id" >> ./lint_message

lint_num=`wc -l ./lint_message | awk '{print $1}'`
if [ $lint_num ]; then
        cat ./lint_message
        echo "LINT FAILED!" >> ./lint_message
fi
