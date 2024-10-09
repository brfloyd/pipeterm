#!/usr/bin/env bash

while true; do 
  pkill -f 'go run main.go'
  go run main.go &
  inotifywait -e attrib $(find . -name '*.go')  || exit
done
