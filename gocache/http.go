/*
 * @Author: wangqian
 * @Date: 2025-02-11 15:53:10
 * @LastEditors: wangqian
 * @LastEditTime: 2025-02-14 19:36:14
 */
/*
分布式缓存需要实现节点之间的通信,暂时使用基于HTTP来实现通信
如果一个节点启动HTTP服务，那么这个节点就可以被其他节点访问
TODO:使用rpc方式来实现节点之间的通信
*/
package gocache

import (
	"fmt"
	"goCache/gocache/consistenthash"
	"io/ioutil"
	"sync"

	// "go/format"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// 先创建一个结构体 承载节点之间HTTP通信的核心数据结构

// 定义默认路径
const (
	defaultBasePath = "/gocache/"
	defaultReolicas = 50
)

// 实现服务端
type HTTPPool struct {
	// self用来记录自己的地址，包括主机名/ip和端口
	self string
	// 节点通讯地址前缀
	basepath string
	mu       sync.Mutex
	// hash算法 通过具体key来选择节点
	peers *consistenthash.Map
	// 映射远程节点对应的httpgetter，每个远程节点对应一个httpgetter
	httpGetters map[string]*httpGetter
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

// 实例化一致性哈希算法，并添加传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReolicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	// 并为每一个节点创建了一个HTTP客户端httpGetter
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.basepath,
		}
	}
}

// 包装了一致性哈希算法中的get方法，并根据传入的key返回对应的http客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// 实现HTTP客户端 httpgetter
type httpGetter struct {
	baseURL string
}

// 传入节点要节点名称和key 通过fmt库来实现字符串的格式化
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned :%v", res.Status)
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body:%v", err)
	}
	return bytes, nil
}

// 断言
var _ PeerGetter = (*httpGetter)(nil)
