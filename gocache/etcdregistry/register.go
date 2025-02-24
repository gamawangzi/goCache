/*
 * @Author: wangqian
 * @Date: 2025-02-21 15:41:18
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-24 19:12:48
 */
package etcdregistry

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"

)

// etcd默认配置
var (
	defaultEtcdConfig = clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
)



// 注册服务到etcd，并保持心跳检测 （Raft）
func Register(service string, addr string, stop chan error) error {
	// 创建一个etcd client
	log.Println("this is reister func ")
	cli, err := clientv3.New(defaultEtcdConfig)
	if err != nil {
		return fmt.Errorf("create etcd client failed : %v", err)
	}
	defer cli.Close()
	// 创建一个租约 配置5秒过期
	// 租约用于管理键值对的生命周期，为键值对设置一个过期时间 租约到期自动删除对应的键值对
	resp, err := cli.Grant(context.Background(), 30)
	if err != nil {
		return fmt.Errorf("create lease failed: %v", err)
	}
	leaseId := resp.ID //获取了该租约的 ID
	// 注册服务
	err = etcdAdd(cli, leaseId, service, addr)
	if err != nil {
		return fmt.Errorf("add etcd record failed: %v", err)
	}
	// 设置服务心跳检测,创建了一个保持租约活动的心跳通道 ch，确保租约在生命周期内保持有效。
	ch, err := cli.KeepAlive(context.Background(), leaseId)
	if err != nil {
		return fmt.Errorf("set keepalive failed: %v", err)
	}
	log.Println("this is register func serverice is "+service)
	log.Printf("[%s] register service ok\n", addr)
	for {
		select {
		case err := <-stop:
			if err != nil {
				log.Println(err)
			}
			return err
		case <-cli.Ctx().Done():
			log.Println("service closed")
			return nil
		case _, ok := <-ch:
			// 监听心跳
			if !ok {
				log.Println("keep alive channel closed")
				_, err := cli.Revoke(context.Background(), leaseId)
				return err
			}
			//log.Printf("Recv reply from service: %s/%s, ttl:%d", service, addr, resp.TTL)
		}
	}
}

// 输入参数分别为etcd客户端，etcd租约ID，服务名称，服务地址
func etcdAdd(c *clientv3.Client, leaseId clientv3.LeaseID, service, addr string) error {
	em, err := endpoints.NewManager(c, service) //创建一个用于管理 etcd 中的服务端点（endpoints）
	if err != nil {
		return err
	}
	//该方法用于将指定的服务地址（addr）添加到 etcd 中的服务端点列表中。
	//clientv3.WithLease(lid) 选项表示使用指定的租约 ID（lid）来设置键值的生命周期。
	//如果添加服务地址成功，函数会返回 nil 表示没有错误；如果发生错误，函数会返回相应的错误信息
	return em.AddEndpoint(c.Ctx(), service+"/"+addr, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(leaseId))
}
