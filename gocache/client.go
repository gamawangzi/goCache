package gocache

import (
	"context"
	"fmt"
	pb "goCache/gocache/gocachepb"
	"log"

	"google.golang.org/grpc"
)

//实现grpc客户端
type client struct{
	// 名称
	name string 
}
// 从远程节点获取对应缓存值 
func (c *client) Get(group string,key string)([]byte,error){
	// 先根据group和key获取到对应的访问路径 
	addr := fmt.Sprintf(
		"%v%v/%v",
		c.name,
		group,
		key,
	)
	// 获取grpc连接，并设置链接参数为不加密以及阻塞等待
	conn,err := grpc.Dial(addr,grpc.WithInsecure(),grpc.WithBlock())
	if err != nil{
		log.Fatalf("connect error ! ")
	}
	defer conn.Close()
	grpcClient := pb.NewGroupCacheClient(conn)

	// 为grpc远程调用设置超时时间 
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := grpcClient.Get(ctx, &pb.Request{
		Group: group,
		Key:   key,
	})
	if err != nil{
		return nil, fmt.Errorf("can not get %s/%s from peer %s", group, key, c.name)
	}
	return resp.GetValue(),nil
}
func NewClient(service string)*client{
	return &client{name:service}
}
// 进行断言 
var _ PeerGetter = (*client)(nil)