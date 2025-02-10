/*
 * @Author: wangqian
 * @Date: 2025-02-09 15:35:55
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-10 17:19:23
 */
// package main
package gocache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"key1":"11",
	"key2":"22",
	"key3":"33",
}

// 测试group实例 
func TestGet(t *testing.T) {
	loadCounts := make(map[string]int,len(db))
	mycache := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	for k,v := range db{
		// 第一次缓存中不存在 走回调函数 
		if view,err := mycache.Get(k);err != nil || view.String() != v{
			t.Fatal("faile to get value of key1")
		}
		// 第二次直接从缓存中获取 如果loadcounts大于1表示没有从缓存中获取导致测试失败 
		if _, err := mycache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} 
	}
	if view, err := mycache.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
	
}