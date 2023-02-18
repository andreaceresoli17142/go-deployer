#!/bin/bash

go mod init go-deployer 2> /dev/null || :
go mod tidy 2> /dev/null || :
go build . 

killall -ew ./go-deployer || : 
./go-deployer &
