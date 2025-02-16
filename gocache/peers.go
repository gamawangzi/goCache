/*
 * @Author: wangqian
 * @Date: 2025-02-13 16:37:25
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-16 16:09:30
 */
package gocache
import pb "goCache/gocache/gocachepb/gocachepb"

type PeerPicker interface {
	// 根据传入的key选择响应的节点 getter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 通过get来从对应group中查找缓存值
	// Get(group string,key string)([]byte,error)
	Get(in *pb.Request, out *pb.Response) error
}
