/*
 * @Author: wangqian
 * @Date: 2025-02-17 14:51:26
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-24 19:11:05
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
func NewServer(self string) (*Server, error) {
	return &Server{
		self:    self,
		peers:   consistenthash.New(defaultgrpcReolicas, nil),
		clients: map[string]*Client{},
	},nil
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
		return fmt.Errorf("server already started")
	}
	// -----------------启动服务----------------------
	// 1. 设置status为true 表示服务器已在运行
	// 2. 初始化stop channel,这用于通知registry stop keep alive
	// 3. 初始化tcp socket并开始监听
	// 4. 注册rpc服务至grpc 这样grpc收到request可以分发给server处理
	// 5. 将自己的服务名/Host地址注册至etcd 这样client可以通过etcd
	//    获取服务Host地址 从而进行通信。这样的好处是client只需知道服务名
	//    以及etcd的Host即可获取对应服务IP 无需写死至client代码中
	// ----------------------------------------------
	p.status = true
	p.stopSignal = make(chan error)

	port := strings.Split(p.self, ":")[1]
	lis, err := net.Listen("tcp", ":"+port) //监听指定的 TCP 端口，用于接受客户端的 gRPC 请求
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	gpb.RegisterGroupCacheServer(grpcServer, p)
	//创建一个新的 gRPC 服务器 grpcServer，然后将当前的 Server 对象 s 注册为 gRPC 服务。
	//这样，gRPC 服务器就能够处理来自客户端的请求。

	go func() {
		// 注册服务至 etcd。该操作会一直阻塞，直到停止信号被接收。
		//当停止信号被接收后，关闭通知通道 s.stopSignal，关闭 TCP 监听端口，并输出日志表示服务已经停止。
		err := etcdregistry.Register("gocache", p.self, p.stopSignal)
		if err != nil {
			log.Fatalf(err.Error())
		}
		// Close channel
		close(p.stopSignal)
		// Close tcp listen
		err = lis.Close()
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.Printf("[%s] Revoke service and close tcp socket ok.", p.self)
	}()

	p.mu.Unlock()

	//启动 gRPC 服务器。grpcServer.Serve(lis) 会阻塞，处理客户端的 gRPC 请求，直到服务器关闭或发生错误。
	//如果服务器状态为运行状态（s.status 为 true），并且发生了错误，则返回相应的错误。
	if err := grpcServer.Serve(lis); p.status && err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

// 实现get接口 用于处理grpc客户端请求 
func (p *Server) Get(ctx context.Context, in *gpb.Request) (*gpb.Response, error) {
	// 和http一样 先获取到需要groupname和key
	group, key := in.Group, in.Key
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
		service := fmt.Sprintf("gocache/%s", peer) 
		p.clients[peer] = NewClient(service)
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
