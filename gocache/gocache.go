/*
 * @Author: wangqian
 * @Date: 2025-02-10 15:42:05
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-19 19:10:29
 */
package gocache

import (
	"fmt"
	"goCache/gocache/singleflight"
	"log"
	"sync"
	//  pb "goCache/gocache/gocachepb/gocachepb"
	pb "goCache/gocache/gocachepb"
)

/*
	如果缓存不存在，需要从数据源获取缓存数据添加到缓存中
	所以设计一个回调函数，如果缓存不存在就调用回调函数得到数据
*/

// 定义接口和回调函数
type Getter interface{
	Get(key string) ([]byte,error)
}

type GetterFunc func (key string) ([]byte,error)

func (f GetterFunc )Get(key string)([]byte,error){
	return f(key)
}

// 定义group
/*
	一个group可以认为是一个缓存的命名空间，每个group都拥有一个唯一的name
	getter表示缓存未命中是获取数据的回调函数 
	maincache实现并发缓存 
*/
type Group struct{
	name string
	getter Getter
	mainCache cache
	peers PeerPicker
	
	// 防止缓存击穿 
	loader *singleflight.Group
}

var (
	mu sync.RWMutex
	groups = make(map[string]*Group)
)
// 实现new函数 
func NewGroup(name string,cacheBytes int64,getter Getter)*Group{
	if getter == nil{
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name: name,
		getter: getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader: &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// 获取到group
func GetGroup(name string)*Group{
	// 只读锁 
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}
// 核心方法 通过key来获取到缓存中的value
func (g *Group)Get(key string)(ByteView,error){
	if key == ""{
		return ByteView{},fmt.Errorf("key is required")
	}
	if v,ok := g.mainCache.get(key);ok{
		log.Println("cache get")
		return v,nil
	}
	// 如果缓存没有命中，则调用local方法 
	return g.load(key)
}
// 注册节点 
func (g *Group)RegisterPeers(peers PeerPicker){
	if g.peers != nil{
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}
// 选择调用节点 
func (g *Group)load(key string)(value ByteView,err error){
	// if g.peers != nil{
	// 	if peer,ok := g.peers.PickPeer(key);ok{
	// 		if value,err = g.getFromPeer(peer,key);err == nil{
	// 			return value,nil
	// 		}
	// 		log.Println("[gocache] Failed to get from peer", err)
	// 	}
	// }
	// // 失败调用回调函数 
	// return g.getLocally(key)
	// 防止缓存击穿 使用do函数 
	viewi,err := g.loader.Do(key,func() (interface{}, error) {
		if g.peers != nil{
			if peer,ok := g.peers.PickPeer(key);ok{
				if value,err = g.getFromPeer(peer,key);err == nil{
					return value,nil
				}
				log.Println("[gocache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil{
		return viewi.(ByteView),nil
	}
	return 
}

func (g *Group)getFromPeer(peer PeerGetter,key string)(ByteView,error){
	// bytes,err := peer.Get(g.name,key)
	// if err != nil{
	// 	return ByteView{},err
	// }
	// return ByteView{b:bytes},nil
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	// res := &pb.Response{}
	value,err := peer.Get(req.Group, req.Key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: value}, nil
}
func (g *Group)getLocally(key string)(ByteView,error){
	// 调用回调方法来获取到数据源
	bytes,err := g.getter.Get(key)
	if err != nil{
		return ByteView{},err
	}
	value := ByteView{b: cloneBytes(bytes)}
	// 然后调用方法把key和value传入到缓存中 
	g.populateCache(key,value)
	return value,nil
}
func (g *Group)populateCache(key string,value ByteView){
	g.mainCache.add(key,value)
}


