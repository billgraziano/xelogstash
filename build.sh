#!/bin/bash
if [ $# -eq 0 ]; then
    echo "No version provided"
    exit 1
fi
echo "Building..."
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_TAG=$(git describe --tags --dirty --always)
BLD_TIME=$(date +%FT%T%Z)
VERSION=$1
echo "GIT_COMMIT: $GIT_COMMIT"
echo "GIT_TAG:    $GIT_TAG"
echo "BLD_TIME:   $BLD_TIME"
echo "VERSION:    $VERSION"

DEPLOY=./deploy
TARGET=$DEPLOY/linux/sqlxewriter

echo ""
echo "DEPLOY:     $DEPLOY"
echo "TARGET:     $TARGET"

rm -rf $TARGET 
mkdir -p $TARGET

go build -o $TARGET/sqlxewriter.exe -ldflags "-s -w -X main.version=$VERSION -X main.sha1ver=$GIT_COMMIT -X main.buildTime=$BLD_TIME -X main.builtBy=buildsh" ./cmd/sqlxewriter
cp -r ./samples $TARGET
cp ./LICENSE.txt $TARGET
cp ./README.html $TARGET
cp ./samples/sqlxewriter.toml $TARGET

TAR_FILE=$DEPLOY/sqlxewriter_${GIT_TAG}_linux_x64.tar
rm $TAR_FILE 
echo "Making ${TAR_FILE}..."
tar -cf $TAR_FILE $TARGET
tar -tf $TAR_FILE
