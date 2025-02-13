package gocache


type PeerPicker interface {
	// 根据传入的key选择响应的节点 getter
	PickPeer(key string) (peer PeerGetter,ok bool)
}

type PeerGetter interface{
	// 通过get来从对应group中查找缓存值
	Get(group string,key string)([]byte,error)
}

