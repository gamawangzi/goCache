/*
 * @Author: wangqian
 * @Date: 2025-02-09 15:52:37
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-09 16:30:14
 */
package lru

import "container/list"

// 使用lru淘汰策略
type Cache struct{
	// 允许使用的最大内存
	maxBytes int64
	// 当前已经使用的内存 
	nbytes int64
	// 双向链表 lru
	ll *list.List
	// cache键值对
	cache map[string]*list.Element
	// 记录移除时的回调函数 
	OnEvicted func(key string,value Value)
}
// 这是双向链表中的节点数量类型，在链表中也保存了对应的key，
// 为了方便在淘汰队首节点时，需要通过key删除对应的映射 
type entry struct{
	key string
	value Value
}
// 使用len函数来记录它携带了多少字节 
type Value interface{
	Len() int
}

// 实现cache new函数 
func New(maxBytes int64,onEvicted func(string,Value)) *Cache{
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找 及缓存命中 首先需要在map中找到对应双向链表的节点 然后将节点移动到链表的末尾
func (c *Cache) Get(key string)(value Value,ok bool){
	if ele,ok := c.cache[key];ok{
		// 找到了对应的节点 这里约定front为队尾 
		c.ll.MoveToFront(ele)
		// 获取到对应键值对 
		kv := ele.Value.(*entry)
		return kv.value,true
	}
	return 
}
// 缓存淘汰 即淘汰最近最少访问的节点 
func(c *Cache)RemoveOldest(){
	// 先回去到最后一个元素
	ele := c.ll.Back()
	if ele != nil{
		// 删除 
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 删除map中key对应的键值对
		delete(c.cache,kv.key)
		// 然后改变当前所拥有的字节数 删除key和value对应的字节数 
		c.nbytes = c.nbytes - int64(len(kv.key)) - int64(kv.value.Len())
		// 如果删除缓存的回调函数存在就要执行对应的回调函数 
		if c.OnEvicted!=nil{
			c.OnEvicted(kv.key,kv.value)
		}
	}
}
// 修改或新增缓存 
func(c *Cache) Add(key string,value Value){
	// 如果当前缓存已经存在 即表示修改缓存 
	if ele,ok := c.cache[key];ok{
		// 先移动到队首
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 改变当前字节
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	}else{
		// 新增 
		ele := c.ll.PushFront(&entry{
			key: key,
			value: value,
		})
		c.cache[key] = ele
		c.nbytes += int64(value.Len()) + int64(len(key))
	}
	// 如果当前的最大字节容量已经小于当前容量那么就要淘汰 为0表示不做限制
	for c.maxBytes!=0 && c.maxBytes < c.nbytes{
		c.RemoveOldest()
	}
}
//测试 
func(c *Cache)Len() int{
	return c.ll.Len()
}