#!/bin/bash

version=$( cat version/VERSION )

rm -rf release && mkdir release
cp cadvisor release/cadvisor
go get -u github.com/progrium/gh-release
gh-release create google/cadvisor ${version}
