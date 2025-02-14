/*
 * @Author: wangqian
 * @Date: 2025-02-11 16:19:30
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-14 16:53:12
 */
package main

import (
	"flag"
	"fmt"
	"goCache/gocache"

	// "gocache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建缓存group
func createGroup() *gocache.Group{
	return gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}
// 创建缓存服务器，创建 HTTPPool，添加节点信息，注册到cache中，并启动HTTP服务一共三个端口
func startCacheServer(addr string,addrs []string,cache *gocache.Group){
	peers := gocache.NewHTTPPool(addr)
	peers.Set(addrs...)
	cache.RegisterPeers(peers)
	log.Println("gocache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}
// 启动一共API服务，端口9999，与用户进行交互 
func startAPIServer(apiAddr string,cache *gocache.Group){
	http.Handle("/api",http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view,err := cache.Get(key)
			if err != nil{
				http.Error(w,err.Error(),http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type","application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at ",apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:],nil))
}
func main() {
	var port int
	var api bool
	// 通过命令行来实现参数传入 
	flag.IntVar(&port, "port", 8001, "GoCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()
	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	cache := createGroup()
	if api {
		go startAPIServer(apiAddr, cache)
	}
	startCacheServer(addrMap[port], []string(addrs), cache)
}
