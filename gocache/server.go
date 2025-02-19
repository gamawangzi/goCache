/*
 * @Author: wangqian
 * @Date: 2025-02-17 14:51:26
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-19 19:56:45
 */
package gocache

import (
	"context"
	"fmt"
	"goCache/gocache/consistenthash"
	gpb "goCache/gocache/gocachepb"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// 使用grpc方式来实现节点之间的通信

const (
	defaultgrpcBasePath = "/gocache/"
	defaultgrpcReolicas = 50
)

type server struct {
	gpb.UnimplementedGroupCacheServer
	// 其他参数和http一样
	self     string
	basepath string
	mu       sync.Mutex
	// hash算法
	peers *consistenthash.Map
	// TODO:实现一个grpc的client 并用map进行映射
	clients map[string]*client
}

// 实现Server的new函数
func NewServer(self string) *server {
	return &server{
		self:     self,
		basepath: defaultgrpcBasePath,
	}
}
func (p *server) Log(format string, v ...interface{}) {
	log.Printf("[GrpcServer %s] %s", p.self, fmt.Sprintf(format, v...))
}

// grpc实现Start接口 传入apiAddr
func (p *server) Start(apiAddr string) error {
	p.mu.Lock()
	lis, err := net.Listen("tcp", apiAddr)
	if err != nil {
		return fmt.Errorf("failed to listen : %v", err)
	}
	grpcServer := grpc.NewServer()
	gpb.RegisterGroupCacheServer(grpcServer, p)
	    // 注册反射服务
	reflection.Register(grpcServer)
	p.mu.Unlock()
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve :%v", err)
	}
	return nil
}

// 实现get接口
func (p *server) Get(ctx context.Context, in *gpb.Request) (*gpb.Response, error) {
	// 和http一样 先获取到需要groupname和key
	group, key := in.GetGroup(), in.GetKey()
	// 定义返回值
	resp := &gpb.Response{}
	log.Printf("[gocache_svr %s] Recv RPC Request - (%s)/(%s)", p.self, group, key)
	// 有了group的name就可以获取到对应的缓存group
	g := GetGroup(group)
	if g == nil {
		return resp, fmt.Errorf("No this group")
	}
	// 接着获取对应的值
	view, err := g.Get(key)
	if err != nil {
		return resp, err
	}
	resp.Value = view.ByteSlice()
	return resp, nil
}

// 实现set方法 实例化hash算法 并添加传入的节点
func (p *server) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultgrpcReolicas, nil)
	p.peers.Add(peers...)
	// TODO:
	p.clients = make(map[string]*client)
	for _,peer := range peers{
		p.clients[peer] = NewClient(peer + p.basepath)
	}
}

// 实现http.go中对应的pickpeer方法 
// TODO:解决hash映射到远程节点调用没有返回值问题
func (p *server) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		// TODO:
		return p.clients[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*server)(nil)
