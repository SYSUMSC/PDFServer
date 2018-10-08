#!/usr/bin/env bash

set -e

if [ ! -f build.sh ]; then
	echo 'build must be run within its container folder' 1>&2
fi

go build server.go

echo 'finished'
