#!/bin/bash

version=$( cat version/VERSION )
branch=$( git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown' )

rm -rf release && mkdir release
cp cadvisor release/cadvisor
go get -u github.com/progrium/gh-release
gh-release create google/cadvisor ${version} ${branch} ${version}
