#!/bin/bash
if [ $# -eq 0 ]; then
    echo "No version provided"
    exit 1
fi
echo "Building..."
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_TAG=$(git describe --tags --dirty --always)
BLD_TIME=$(date +%FT%T%:z)
VERSION=$1
# echo "GIT_COMMIT: $GIT_COMMIT"
echo "GIT_TAG:    $GIT_TAG"
echo "BLD_TIME:   $BLD_TIME"
echo "VERSION:    $VERSION"

DEPLOY=./deploy
PARENT=$DEPLOY/linux
TARGET=$PARENT/sqlxewriter

echo ""
echo "DEPLOY:     $DEPLOY"
echo "PARENT:     $PARENT"
echo "TARGET:     $TARGET"

rm -rf $TARGET 
mkdir -p $TARGET
echo "Running go vet and go test..."
go vet -all ./cmd/sqlxewriter
go vet -all ./config ./log ./logstash ./seq ./status ./summary ./xe ./sink ./pkg/...
go test ./cmd/xelogstash ./cmd/sqlxewriter ./config ./seq ./xe ./sink ./status ./pkg/...

echo "Building sqlxewriter.exe..."
go build -o $TARGET/sqlxewriter.exe -ldflags "-s -w -X main.version=$VERSION -X main.sha1ver=$GIT_TAG -X main.buildTime=$BLD_TIME -X main.builtBy=buildsh" ./cmd/sqlxewriter
./deploy/linux/sqlxewriter/sqlxewriter.exe -version 

cp -r ./samples $TARGET
cp ./LICENSE.txt $TARGET
cp ./README.html $TARGET
cp ./samples/sqlxewriter.toml $TARGET

TAR_FILE=$DEPLOY/sqlxewriter_${GIT_TAG}_linux_x64.tar.gz
rm -f $TAR_FILE 
echo "Making ${TAR_FILE}..."
tar -C $PARENT -czf $TAR_FILE sqlxewriter
# tar -tf $TAR_FILE
