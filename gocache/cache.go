/*
 * @Author: wangqian
 * @Date: 2025-02-10 15:31:13
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-10 15:35:33
 */
package gocache

import (
	// "goCache/lru"
	"goCache/gocache/lru"
	"sync"
)

// 封装一层lru中的cache 从而实现支持并发读写 并封装add和get方法
type  cache struct{
	mu sync.Mutex
	lru *lru.Cache
	cacheBytes int64
}

func (c *cache)add(key string,value ByteView){
	// 上锁 
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil{
		// 如果当前缓存为空则new一个 延迟初始化 提高性能 减少要求 
		c.lru = lru.New(c.cacheBytes,nil) 
	}
	// 然后添加缓存 
	c.lru.Add(key,value)
}

func(c *cache)get(key string)(value ByteView,ok bool){
	// 上锁 
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil{
		return
	}
	if v,ok := c.lru.Get(key);ok{
		return v.(ByteView),ok
	}
	return
}
