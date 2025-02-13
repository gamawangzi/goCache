package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	hash := New(3, func(data []byte) uint32 {
		i, _ := strconv.Atoi(string(data))
		return uint32(i)
	})
	hash.Add("6","4","2")
	// 看是否和预想的一致 虚拟节点对应的真实节点 
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}
	for k,v := range testCases{
		if hash.Get(k) != v{
			t.Errorf("Asking for %s,should have yielded %s",k,v)
		}
	}
	hash.Add("8")
	testCases["27"] = "8"
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
