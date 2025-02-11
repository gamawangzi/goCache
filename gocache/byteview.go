package gocache

// 表示缓存值

type ByteView struct{
	// 支持任意数据类型的存储
	b []byte
}
//实现需要的函数 在lru cache中定义了value接口需要实现Len函数 
func (v ByteView)Len() int{
	return len(v.b)
}
// 复制缓存为一个byte切片 
func (v ByteView)ByteSlice() []byte{
	return cloneBytes(v.b)
}
// 转化为string
func (v ByteView)String() string{
	return string(v.b)
}
// b是只读的 防止被修改 
func cloneBytes(b []byte)[]byte{
	c := make([]byte,len(b))
	copy(c,b)
	return c
}
