package lfu

import "testing"

type String string

func (d String) Len() int {
	return len(d)
}
func TestGet(t *testing.T) {
	lfu := New(int64(0), nil, 60)
	//在这个特定的上下文中，int64(0) 作为参数传递给 New 函数，用于指定 LRU 缓存的最大存储容量。
	//在这里，将其设置为 0 表示缓存的最大容量为零，即没有存储空间，因此不会保存任何键值对。
	//这可以用于创建一个非常小的缓存或用于特定的测试场景，其中不需要实际存储数据。
	lfu.Add("key1", String("1234"), 60)
	if v, ok := lfu.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lfu.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}
