/*
 * @Author: wangqian
 * @Date: 2025-02-10 15:42:05
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-24 16:56:29
 */
package gocache

import (
	"fmt"
	"goCache/gocache/singleflight"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	//  pb "goCache/gocache/gocachepb/gocachepb"
	pb "goCache/gocache/gocachepb"
)

/*
	如果缓存不存在，需要从数据源获取缓存数据添加到缓存中
	所以设计一个回调函数，如果缓存不存在就调用回调函数得到数据
*/

// 定义接口和回调函数
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 定义group
/*
	一个group可以认为是一个缓存的命名空间，每个group都拥有一个唯一的name
	getter表示缓存未命中是获取数据的回调函数
	maincache实现并发缓存
*/
type Group struct {
	// 缓存名称
	name string
	// 从数据源获取数据
	getter Getter
	// 主缓存
	mainCache cache
	// 热门数据缓存
	hotCache cache
	// 用于根据key来选择响应的缓存节点
	peers PeerPicker
	// 防止缓存击穿
	loader *singleflight.Group
	// key的统计信息
	keys map[string]*KeyStats
}

// 通过封装原子类 来实现请求次数的统计 保证并发安全
type AtomicInt int64

// Add 方法用于对 AtomicInt 中的值进行原子自增
func (i *AtomicInt) Add(n int64) { //原子自增
	atomic.AddInt64((*int64)(i), n)
}

// Get 方法用于获取 AtomicInt 中的值。
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

type KeyStats struct { //Key的统计信息
	firstGetTime time.Time //第一次请求的时间
	remoteCnt    AtomicInt //请求的次数（利用atomic包封装的原子类）
}

var (
	// 最大QPS
	maxQPS = 10
	// 读写锁
	mu sync.RWMutex
	// 根据缓存组的名字来获取对应的缓存组
	groups = make(map[string]*Group)
)

// 实现new函数
// TODO: 实现传入不同参数达到不同的淘汰算法 LRU LFU
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
		keys:      map[string]*KeyStats{},
		hotCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// 获取到group
func GetGroup(name string) *Group {
	// 只读锁
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 核心方法 通过key来获取到缓存中的value
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 两张cache 先查看hotcache中有没有对应的缓存
	if v, ok := g.hotCache.get(key); ok {
		log.Println("hotCache get")
		return v, nil
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("maincache get")
		return v, nil
	}
	// 如果缓存没有命中，则调用local方法
	return g.load(key)
}

// 注册节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 选择调用节点
func (g *Group) load(key string) (value ByteView, err error) {
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
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[gocache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// bytes,err := peer.Get(g.name,key)
	// if err != nil{
	// 	return ByteView{},err
	// }
	// return ByteView{b:bytes},nil
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	// log.Println("this is getFromPeer func ")
	res := &pb.Response{}
	// res := &pb.Response{}
	log.Println("this is getFromPeer func ")
	err := peer.Get(req, res)
	
	if err != nil {
		log.Fatal("ERROR",err)
		return ByteView{}, err
	}
	// 计算QPS
	if stat, ok := g.keys[key]; ok {
		stat.remoteCnt.Add(1)
		interval := float64(time.Now().Unix()-stat.firstGetTime.Unix()) / 60
		qps := stat.remoteCnt.Get() / int64(math.Max(1, math.Round(interval)))
		if qps >= int64(maxQPS) {
			// 存入hotcache中
			g.populateHotCache(key, ByteView{b: res.Value})
			//删除映射关系,节省内存
			mu.Lock()
			delete(g.keys, key)
			mu.Unlock()
		} else {
			// 第一次
			g.keys[key] = &KeyStats{
				firstGetTime: time.Now(),
				remoteCnt:    1,
			}
		}
	}
	return ByteView{b: res.Value}, nil
}
func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用回调方法来获取到数据源
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	// 然后调用方法把key和value传入到缓存中
	g.populateCache(key, value)
	return value, nil
}
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// populateHotCache 将数据添加到hotCache中
func (g *Group) populateHotCache(key string, value ByteView) {
	g.hotCache.add(key, value)
}
