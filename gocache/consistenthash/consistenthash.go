package consistenthash

import (
	// "fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

//实现一致性hash
/*
	一致性hash通常将key映射到2^32的空间中 形成一个环
	然后通过map映射虚拟节点，从而解决数据倾斜问题
*/

// 可以自定义hash函数
type Hash func(data []byte)uint32

type Map struct{

	hash Hash
	// 虚拟节点的倍数 
	replicas int
	// 哈希环
	keys []int
	// 虚拟节点和真实节点的映射表 key是虚拟节点的hash值 值是真实节点的名称 
	hashmap map[int]string
}
// 允许自定义虚拟节点倍数和hash函数 
func New(replicas int,fn Hash)*Map{
	m := &Map{
		replicas: replicas,
		hash: fn,
		hashmap: make(map[int]string),
	}
	if m.hash == nil{
		m.hash = crc32.ChecksumIEEE
	}
	return m
}
// 实现添加真实节点的add方法 允许传入0或者多个真实节点的名称 
func (m *Map) Add(keys ...string){
	for _,key := range keys{
		// 一个真实节点创造多个虚拟节点
		for i:=0;i < m.replicas;i++{
			//虚拟节点名称是strconv.Itoa(i)+key 拼接起来01，11，21类似于
			hash := int(m.hash([]byte(strconv.Itoa(i)+key)))
			m.keys = append(m.keys, hash)
			// fmt.Println(key)
			// 真实节点和虚拟节点映射 
			m.hashmap[hash] = key
		}
	}
	// 环上的哈希值排序 
	sort.Ints(m.keys)
}

//  实现选择节点的get方法 
func (m *Map)Get(key string)string{
	if len(m.keys) == 0{
		return ""
	}
	hash := int(m.hash([]byte(key)))
	// 找到第一个节点 
	index := sort.Search(len(m.keys),func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashmap[m.keys[index%len(m.keys)]]
}