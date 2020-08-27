#!/bin/bash
echo "Building..."
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_TAG=$(git describe --tags --dirty --always)
BLD_TIME=$(date +%FT%T%Z)
echo "GIT_COMMIT: $GIT_COMMIT"
echo "GIT_TAG:    $GIT_TAG"
echo "BLD_TIME:   $BLD_TIME"
TAR_FILE=./dist/sqlxewriter_${GIT_TAG}_linux_amd64.tar
mkdir -p ./dist/sqlxewriter_linux_amd64
go build -o ./dist/sqlxewriter_linux_amd64/sqlxewriter -ldflags "-s -w -X main.version=$GIT_TAG -X main.sha1ver=$GIT_COMMIT -X main.buildTime=$BLD_TIME -X main.builtBy=buildsh" ./cmd/sqlxewriter
cp -r ./samples ./dist/sqlxewriter_linux_amd64
cp ./LICENSE.txt ./dist/sqlxewriter_linux_amd64
cp ./README.html ./dist/sqlxewriter_linux_amd64
echo "Making ${TAR_FILE}..."
rm $TAR_FILE
tar -cf $TAR_FILE ./dist/sqlxewriter_linux_amd64
tar -tf $TAR_FILE
