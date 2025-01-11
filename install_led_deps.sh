#!/bin/bash

git clone https://github.com/jgarff/rpi_ws281x.git
cd rpi_ws281x/
scons

sudo cp *.h /usr/include
sudo cp *.a /usr/lib

ls /usr/include/ws2811.h
export CGO_LDFLAGS="-lm"
go clean -modcache
go build