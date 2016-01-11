#!/bin/bash

TAG=`git describe --exact-match --tags 2>/dev/null`
BRANCH=`git rev-parse --abbrev-ref HEAD`

if [ "$BRANCH" == "master" ]; then
  TAG=canary
elif [ "$BRANCH" != "HEAD" ]; then
  TAG=$BRANCH
fi

echo $TAG
