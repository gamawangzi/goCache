/*
 * @Author: wangqian
 * @Date: 2025-02-09 16:33:02
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-09 17:09:30
 */
package lru

import (
	// "goCache/lru"
	"reflect"
	"testing"
)

type String string
// 这里是为了实现value接口中定义的Len函数 从而可以传入对应的值
func (d String) Len() int {
	return len(d)
}
// 测试get函数
func TestGet(t *testing.T) {
	lru := New(int64(0),nil)
	// 添加kv
	lru.Add("key1",String("value1"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "value1" {
		t.Fatalf("cache hit key1=value1 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}
// 测试当内存超过设定值 是否会删除节点 
func TestRemoveoldest(t *testing.T){
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + k2 + v1 + v2)
	lru := New(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))
	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

// 测试回调函数 
func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))
	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}
