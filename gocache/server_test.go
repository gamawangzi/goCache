package gocache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func ceateTestServer() (*Group, *server) {
	mysql := map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}

	g := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[Mysql] search key", key)
			if v, ok := mysql[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// // 随机一个端口 避免冲突
	// port := 50000 + r.Intn(100)
	addr := fmt.Sprintf("http://localhost:9999")

	svr := NewServer(addr)
	svr.Set(addr)
	g.RegisterPeers(svr)
	return g, svr
}
func TestServer_GetKey(t *testing.T) {
	g, server := ceateTestServer()
	go func() {
		// 启动服务
		addr := fmt.Sprintf("localhost:9999")
		err := server.Start(addr)
		if err != nil {
			log.Fatal(err)
		}
	}()
	// 测试存在的key
	view, err := g.Get("Jack")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(view.String(), "589") {
		t.Errorf("key exist error ")
	}
	log.Printf("Jack is -> %s",view.String())
	// 测试不存在的key
	_, err = g.Get("Unknown")
	if err != nil {
		if err.Error() != "Unknown not exist" {
			t.Fatal(err)
		} else {
			t.Log(err.Error())
		}
	}
}
