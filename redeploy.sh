#!/bin/bash

killall -ew ./go-deployer || continue

go mod init main || continue
go mod tidy || continue
go build .

./go-deployer &
