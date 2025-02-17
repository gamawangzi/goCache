/*
 * @Author: wangqian
 * @Date: 2025-02-15 15:43:32
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-16 16:40:04
 */
package singleflight

import "sync"
// call代表正在进行中或者已经结束的请求，使用sync.WaitGroup锁避免重入 
type call struct{
	wg sync.WaitGroup //避免重入
	val interface{}
	err error
}
// 管理不同key的请求 
type Group struct{
	mu sync.Mutex
	m map[string]*call
}
// do方法，接收两个参数，第一个参数是key，第二个参数是函数fn
// 针对相同的key，无论do被调用多少次，函数fn都只会调用一次，需要等待fn调用结束了，返回对应返回值和错误
func (g *Group)Do(key string,fn func()(interface{},error)) (interface{},error){
	g.mu.Lock()
	if g.m == nil{
		g.m = make(map[string]*call)
	}
	if c,ok := g.m[key];ok{
		g.mu.Unlock()
		c.wg.Wait() //如果有请求正在进行中，则等待协程结束 
		return c.val,c.err // 请求结束，返回结果 
	}
	c := new(call)
	c.wg.Add(1) //发起请求前加锁
	g.m[key] = c //添加到对应hashmap中，表明key已经有对应的请求在处理
	g.mu.Unlock()

	c.val,c.err = fn() //调用fn，发起请求 
	c.wg.Done() //请求结束 

	g.mu.Lock()
	delete(g.m,key) //请求完毕之后，把对应hashmap值删除 
	g.mu.Unlock()
	
	return c.val,c.err
}
