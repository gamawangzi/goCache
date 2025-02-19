// /*
//  * @Author: wangqian
//  * @Date: 2025-02-11 16:19:30
//  * @LastEditors: wangqian
//  * @LastEditTime: 2025-02-15 15:34:51
//  */
// package main

// import (
// 	"flag"
// 	"fmt"
// 	"goCache/gocache"

// 	// "gocache"
// 	"log"
// 	"net/http"
// )

// var db = map[string]string{
// 	"Tom":  "630",
// 	"Jack": "589",
// 	"Sam":  "567",
// }

// // 创建缓存group
// func createGroup() *gocache.Group{
// 	return gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
// 		func(key string) ([]byte, error) {
// 			log.Println("[SlowDB] search key", key)
// 			if v, ok := db[key]; ok {
// 				return []byte(v), nil
// 			}
// 			return nil, fmt.Errorf("%s not exist", key)
// 		}))
// }
// // 创建缓存服务器，创建 HTTPPool，添加节点信息，注册到cache中，并启动HTTP服务一共三个端口
// func startCacheServer(addr string,addrs []string,cache *gocache.Group){
// 	peers := gocache.NewHTTPPool(addr)
// 	peers.Set(addrs...)
// 	cache.RegisterPeers(peers)
// 	log.Println("gocache is running at", addr)
// 	log.Fatal(http.ListenAndServe(addr[7:], peers))
// }
// // 启动一共API服务，端口9999，与用户进行交互
// func startAPIServer(apiAddr string,cache *gocache.Group){
// 	http.Handle("/api",http.HandlerFunc(
// 		func(w http.ResponseWriter, r *http.Request) {
// 			key := r.URL.Query().Get("key")
// 			view,err := cache.Get(key)
// 			if err != nil{
// 				http.Error(w,err.Error(),http.StatusInternalServerError)
// 				return
// 			}
// 			w.Header().Set("Content-Type","application/octet-stream")
// 			w.Write(view.ByteSlice())
// 		}))
// 	log.Println("fontend server is running at ",apiAddr)
// 	log.Fatal(http.ListenAndServe(apiAddr[7:],nil))
// }
// func main() {
// 	var port int
// 	var api bool
// 	// 通过命令行来实现参数传入
// 	flag.IntVar(&port, "port", 8001, "GoCache server port")
// 	flag.BoolVar(&api, "api", false, "Start a api server?")
// 	flag.Parse()
// 	apiAddr := "http://localhost:9999"
// 	addrMap := map[int]string{
// 		8001: "http://localhost:8001",
// 		8002: "http://localhost:8002",
// 		8003: "http://localhost:8003",
// 	}
// 	var addrs []string
// 	for _, v := range addrMap {
// 		addrs = append(addrs, v)
// 	}

// 	cache := createGroup()
// 	if api {
// 		go startAPIServer(apiAddr, cache)
// 	}
// 	startCacheServer(addrMap[port], []string(addrs), cache)
// }

// 测试grpc
package main

import (
	"fmt"
	"goCache/gocache"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	var mysql = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	// 新建cache实例
	group := gocache.NewGroup("scores", 2<<10, gocache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[Mysql] search key", key)
			if v, ok := mysql[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	// 创建服务实例
	addr := fmt.Sprintf("http://localhost:8001")
	server := gocache.NewServer(addr)
	addr2 := fmt.Sprintf("http://localhost:8002")
	addr3 := fmt.Sprintf("http://localhost:8003")
	server.Set(addr,addr2,addr3)
	group.RegisterPeers(server)
	// 启动服务
	go func() {
		addr := fmt.Sprintf("localhost:9999")
		err := server.Start(addr)
		if err != nil {
			log.Fatal(err)
		}
	}()
	// 通过函数发送几个请求
	
	view, err := group.Get("Tom")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(view.String())

	time.Sleep(time.Second*2)

	view, err = group.Get("Tom")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(view.String())
	stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

    // 等待终止信号
    fmt.Println("Server is running. Press Ctrl+C to stop.")
    <-stop
    fmt.Println("Stopping server...")

}

func GetTomScore(group *gocache.Group, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("get Tom...")
	view, err := group.Get("Tom")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(view.String())
}
