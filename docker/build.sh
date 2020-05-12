#!/bin/bash
# build docker images of btcagent

TITLE=""
BUILD_TYPE="Release"
BUILD_JOBS="$(nproc)"
GIT_DESCRIBE="$(git describe --tag --long)"

while getopts 't:j:d:' c
do
  case $c in
    t) TITLE="$OPTARG" ;;
    d) BUILD_TYPE="$OPTARG" ;;
    j) BUILD_JOBS="$OPTARG" ;;
  esac
done

if [ "x$TITLE" = "x" ] || [ "x$BUILD_TYPE" = "x" ]; then
	echo "Usage: $0 -t <image-title> -b <base-image> -j<build-jobs>"
	echo "Example: $0 -t huobi/btcproxy -d Release -j$(nproc)"
	exit
fi

docker build -t "$TITLE" -f Dockerfile --build-arg BUILD_TYPE="$BUILD_TYPE" --build-arg BUILD_JOBS="$BUILD_JOBS" --build-arg GIT_DESCRIBE="$GIT_DESCRIBE" ../

