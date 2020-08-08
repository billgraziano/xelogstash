@echo off
set GOARCH=amd64
set GOOS=linux
echo Building for sqlxewriter_linux...
go build -o sqlxewriter_linux .\cmd\sqlxewriter 
set GOARCH=
set GOOS=


