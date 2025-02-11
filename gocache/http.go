/*
 * @Author: wangqian
 * @Date: 2025-02-11 15:53:10
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-11 16:30:19
 */
/*
分布式缓存需要实现节点之间的通信,暂时使用基于HTTP来实现通信
如果一个节点启动HTTP服务，那么这个节点就可以被其他节点访问
TODO:使用rpc方式来实现节点之间的通信
*/
package gocache

import (
	"fmt"
	// "go/format"
	"log"
	"net/http"
	"strings"
)

// 先创建一个结构体 承载节点之间HTTP通信的核心数据结构

// 定义默认路径
const defaultBasePath = "/gocache/"

type HTTPPool struct {
	// self用来记录自己的地址，包括主机名/ip和端口
	self string
	// 节点通讯地址前缀
	basepath string
}

// 实现new函数
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basepath: defaultBasePath,
	}
}
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 实现http Handler  包中的ServeHTTP 方法
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 先判断访问路径是否有basepath 如果没有则返回错误
	if !strings.HasPrefix(r.URL.Path, p.basepath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 默认访问路径格式 /basepath/groupname/key 通过函数分为2个部分
	// 即 /gocache/xxx/xxxkey/
	parts := strings.SplitN(r.URL.Path[len(p.basepath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// 获取到groupname
	groupName := parts[0]
	// 获取到需要访问缓存的key
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 标记为字节流 
	w.Header().Set("Content-Type", "application/octet-stream")
	// 返回写入body
	w.Write(view.ByteSlice())
}
