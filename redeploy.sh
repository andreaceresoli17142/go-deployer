#!/bin/bash

go mod init go-deployer 2> /dev/null || continue
go mod tidy 2> /dev/null || continue
go build . 

killall -ew ./go-deployer || continue
./go-deployer &
