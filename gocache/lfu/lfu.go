package lfu

import (
	"container/heap"
	"log"
	"time"
)

/*
	实现LFU算法
*/
// 最不频繁使用
type LFUCache struct{
	// 最大存储容量 
	maxBytes int64
	// 已使用容量 
	nBytes int64
	// 使用最小堆来实现按使用频率排序 
	heap *entryHeap
	// map 
	cache map[string]*entry
	// 删除时的回调函数 
	OnEvicted  func(key string, value Value)
	// 默认过期时间 
	defaultTTL time.Duration
}
type Value interface{
	Len() int
} 
// 实现entry
type entry struct{
	key string 
	value Value
	// 记录访问频率 
	number int 
	// 堆中的索引 
	index int 
	// 节点的过期时间 
	expire time.Time
}
type entryHeap []*entry
func (h entryHeap)Len()int{return len(h)}
func (h entryHeap)Swap(i,j int){
	h[i],h[j] = h[j],h[i]
	h[i].index = i
	h[j].index = j
}
func (h entryHeap)Less(i,j int)bool{
	return h[i].number < h[j].number
}
func (h *entryHeap)Push(x interface{}){
	entry := x.(*entry)
	entry.index = len(*h)
	*h = append(*h, entry)
}
func (h *entryHeap)Pop()(interface{}){
	old := *h 
	n := len(old)
	entry := old[n-1]
	entry.index = -1
	*h = old[:n-1]
	return entry
}
func New(maxBytes int64, onEvicted func(string, Value), defaultTTL time.Duration) *LFUCache {
	return &LFUCache{
		maxBytes:   maxBytes,
		heap:       &entryHeap{},
		cache:      make(map[string]*entry),
		OnEvicted:  onEvicted,
		defaultTTL: defaultTTL,
	}
}
// 实现Get函数 根据key来获取缓存中的值，如果存在entry则将对应节点的freq频率增加 并调用fix函数维护堆
func (c *LFUCache)Get(key string)(value Value,ok bool){
	if ele,ok := c.cache[key];ok{
		// 找到了对应的节点 
		// 1 查看是否过期 
		if ele.expire.Before(time.Now()){
			// 过期删除entry
			c.removeElement(ele)
			log.Printf("The key - %s has expired ",key)
			return nil,false
		}
		// 调用次数++
		ele.number++
		// 调用fix
		heap.Fix(c.heap,ele.index)
		return ele.value,true
	}
	return
}

// 删除频率最低的缓存 
func (c *LFUCache)RemoveOldest(){
	entry := heap.Pop(c.heap).(*entry)
	delete(c.cache,entry.key)
	c.nBytes -= int64(len(entry.key)) + int64(entry.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(entry.key, entry.value)
	}
}
// 实现add函数 插入一个缓存 
func (c *LFUCache)Add(key string,value Value,ttl time.Duration){
	// 如果当前缓存中已经有 
	if ele,ok := c.cache[key];ok{
		ele.number++
		ele.value = value
		ele.expire = time.Now().Add(ttl)
		heap.Fix(c.heap,ele.index)
	}else{
		// 不存在缓存 
		// 1 先初始化一个entry 
		entry := &entry{
			key: key,
			value: value,
			number: 1,
			expire: time.Now().Add(ttl),
		}
		// 然后插入到堆中 
		heap.Push(c.heap,entry)
		c.cache[key] = entry
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	// 如果超过了最大容量就删除最少使用次数的 
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}
// Len 方法返回当前缓存中的记录数量。
func (c *LFUCache) Len() int {
	return len(c.cache)
}
// removeElement 函数删除传入的缓存项。
func (c *LFUCache) removeElement(e *entry) {
	heap.Remove(c.heap, e.index)
	delete(c.cache, e.key)
	c.nBytes -= int64(len(e.key)) + int64(e.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(e.key, e.value)
	}
}