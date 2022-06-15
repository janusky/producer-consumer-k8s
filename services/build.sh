#!/bin/sh

#
# Create image.
#

# DIR="$PWD"
DIR="$(dirname "$0")"

# --build-arg http_proxy=$http_proxy --build-arg https_proxy=$https_proxy
BUILD_DIR=$1
BUILD_IMAGE=$2

cd $DIR

if [ ! -d "$BUILD_DIR" ]; then
    echo "\e[31m$BUILD_DIR does not exist!\e[0m(solve: ./service-a)"
    exit 1
fi
BUILD_DIR_NAME="${BUILD_DIR%"${BUILD_DIR##*[!/]}"}" # extglob-free multi-trailing-/ trim
BUILD_DIR_NAME="${BUILD_DIR_NAME##*/}"  # remove everything before the last /
if [ -z "$BUILD_IMAGE" ]; then
  BUILD_IMAGE=localhost/$BUILD_DIR_NAME:dev
fi

echo "1-Execute build image $BUILD_IMAGE, dir $BUILD_DIR"
docker build --build-arg APP_PATH=$BUILD_DIR --rm -t $BUILD_IMAGE -f Dockerfile .
