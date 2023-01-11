#!/bin/bash

go mod init go-deployer 2> /dev/null || continue
go mod tidy 2> /dev/null || continue
notify-send $(go build . || notify-send "error compiling go-deployer, aborting deployment"; exit 1 ) 

killall -ew ./go-deployer || continue
./go-deployer &
