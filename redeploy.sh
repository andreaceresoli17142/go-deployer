#!/bin/bash

killall -ew ./go-deployer

go mod init main 2>/dev/null
go mod tidy 2>/dev/null
go build .

./go-deployer
