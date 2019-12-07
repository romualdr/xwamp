@ECHO OFF
ECHO Building executable
go build -ldflags="-H windowsgui"