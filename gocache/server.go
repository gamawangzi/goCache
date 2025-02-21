/*
 * @Author: wangqian
 * @Date: 2025-02-17 14:51:26
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-21 16:16:47
 */
package gocache

import (
	"context"
	"fmt"
	"goCache/gocache/consistenthash"
	"goCache/gocache/etcdregistry"
	gpb "goCache/gocache/gocachepb"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/proto"
)

// 使用grpc方式来实现节点之间的通信

const (
	// defaultgrpcBasePath = "/gocache/"
	defaultgrpcReolicas = 50
)

// 配置etcd客户端默认配置
var (
	defaultEtcdConfig = clientv3.Config{
		// etcd服务器地址
		Endpoints: []string{"127.0.0.1:2379"},
		// 建立连接的超时时间为5秒
		DialTimeout: 5 * time.Second,
	}
)

type Server struct {
	// gprc自动生成的代码
	gpb.UnimplementedGroupCacheServer
	// 其他参数和http一样
	self string
	// 加一个状态 true表示running false表示stop
	status bool
	// 接收通知来使服务器停止运行
	stopSignal chan error
	// 互斥锁
	mu sync.Mutex
	// hash算法
	peers *consistenthash.Map
	// TODO:实现一个grpc的client 并用map进行映射
	clients map[string]*Client
}

// 实现Server的new函数
func NewServer(self string) *Server {
	return &Server{
		self:    self,
		peers:   consistenthash.New(defaultgrpcReolicas, nil),
		clients: map[string]*Client{},
	}
}
func (p *Server) Log(format string, v ...interface{}) {
	log.Printf("[GrpcServer %s] %s", p.self, fmt.Sprintf(format, v...))
}

// grpc实现Start接口
/*
	1首先将status设置为true
	2初始化对应的stop channel
	3初始化socket并监听
	4注册rpc服务到grpc，当grpc收到request时可以分发给server处理
	5 将host地址注册到etcd，client就可以etcd来获取host地址从而进行通信
*/
func (p *Server) Start() error {
	p.mu.Lock()
	if p.status == true {
		p.mu.Unlock()
		return fmt.Errorf("the server already started ")
	}
	p.status = true
	p.stopSignal = make(chan error)
	port := strings.Split(p.self, ":")[1]
	// 监听指定的tcp端口 用于接收客户端grpc的请求
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen : %v", err)
	}
	// 创建一个grpc服务器 并将当前server对象注册为grpc服务 
	grpcServer := grpc.NewServer()
	gpb.RegisterGroupCacheServer(grpcServer, p)
	// 注册反射服务
	// reflection.Register(grpcServer)
	go func ()  {
		// TODO:注册服务至etcd，并阻塞等待停止信号的到来 
		err := etcdregistry.Register("gocache",p.self,p.stopSignal)
		if err != nil{
			log.Fatalf(err.Error())
		}
		close(p.stopSignal)
		// 关闭tcp listen
		err = lis.Close()
		if err != nil{
			log.Fatalf(err.Error())
		}
		log.Printf("close tcp ok ")
	}()
	p.mu.Unlock()
	// 启动grpc服务器，grpcServer.Serve(lis) 会阻塞，处理客户端的 gRPC 请求，直到服务器关闭或发生错误
	if err := grpcServer.Serve(lis); p.status&&err != nil {
		return fmt.Errorf("failed to serve :%v", err)
	}
	return nil
}

// 实现get接口 用于处理grpc客户端请求 
func (p *Server) Get(ctx context.Context, in *gpb.Request) (*gpb.Response, error) {
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
	//将获取到的缓存数据序列化为 protobuf 格式，并存储在响应对象的 Value 字段中
	body, err := proto.Marshal(&gpb.Response{Value: view.ByteSlice()})
	if err != nil {
		log.Printf("encoding response body:%v", err)
	}
	resp.Value = body
	return resp, nil
}

// 实现set方法 实例化hash算法 并添加传入的节点
func (p *Server) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers.Add(peers...)
	// TODO:将客户端映射到map中
	for _, peer := range peers {
		p.clients[peer] = NewClient(peer)
	}
}

// 实现http.go中对应的pickpeer方法
// TODO:解决gRPC客户端调用，hash映射到远程节点调用没有返回值问题
func (p *Server) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		// TODO:
		return p.clients[peer], true
	}
	return nil, false
}

func (p *Server)Stop(){
	p.mu.Lock()
	if p.status == false{
		p.mu.Unlock()
		return 
	}
	// 发送停止keepalive信号 
	p.stopSignal <- nil
	// 设置服务运行状态为stop
	p.status = false 
	// 情况消息
	p.clients = nil
	p.clients = nil
	p.mu.Unlock()
}
var _ PeerPicker = (*Server)(nil)
