package gocache

import (
	"context"
	"fmt"
	"goCache/gocache/etcdregistry"
	pb "goCache/gocache/gocachepb"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
	// "google.golang.org/grpc"
)

//实现grpc客户端
type Client struct{
	// 名称
	name string 
}

// 从远程节点获取对应缓存值 
func (c *Client) Get(in *pb.Request, out *pb.Response)(error){
	// 先根据group和key获取到对应的访问路径 
	
	// 创建一个etcd客户端 
	cli,err := clientv3.New(defaultEtcdConfig)
	if err != nil{
		log.Fatalf("connect etcd  error ! ")
	}
	defer cli.Close()
	conn,err := etcdregistry.EtcdDial(cli,c.name)

	if err != nil{
		return err
	}
	defer conn.Close()
	grpcClient := pb.NewGroupCacheClient(conn)

	// 为grpc远程调用设置超时时间 
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := grpcClient.Get(ctx, in)
	if err != nil{
		return fmt.Errorf("can not get %s/%s from peer %s", in.Group,in.Key, c.name)
	}
	if err = proto.Unmarshal(resp.GetValue(), out); err != nil {
		return fmt.Errorf("decoding response body:%v", err)
	}
	return nil
}
func NewClient(service string)*Client{
	return &Client{name:service}
}
// 进行断言 
var _ PeerGetter = (*Client)(nil)