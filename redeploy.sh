#!/bin/bash

go mod init main || continue
go mod tidy || continue
notify-send $(go build . || notify-send("error compiling go-deployer, aborting deployment"); exit 1 ) 

# killall -ew ./go-deployer || continue
# ./go-deployer &
