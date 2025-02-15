#!/bin/bash
###
 # @Author: wangqian
 # @Date: 2025-02-14 16:47:50
 # @LastEditors: wangqian
 # @LastEditTime: 2025-02-15 17:05:05
### 
trap "rm server;kill 0" EXIT

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

wait