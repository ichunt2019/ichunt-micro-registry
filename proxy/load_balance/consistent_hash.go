package load_balance

import (
	"errors"
	"fmt"
	"github.com/syyongx/php2go"
	"hash/crc32"
	"net"
	"sort"
	"strconv"
	"sync"
)

type Hash func(data []byte) uint32

type UInt32Slice []uint32

func (s UInt32Slice) Len() int {
	return len(s)
}

func (s UInt32Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s UInt32Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ConsistentHashBanlance struct {
	mux      sync.RWMutex
	hash     Hash
	lock               sync.Mutex
	replicas int               //复制因子
	keys     map[string]UInt32Slice       //已排序的节点hash切片
	hashMap  map[string]map[uint32]string //节点哈希和Key的map,键是hash值，值是节点key

}

func NewConsistentHashBanlance(replicas int, fn Hash) *ConsistentHashBanlance {
	m := &ConsistentHashBanlance{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[string]map[uint32]string),
	}
	if m.hash == nil {
		//最多32位,保证是一个2^32-1环
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 验证是否为空
func (c *ConsistentHashBanlance) IsEmpty() bool {
	return len(c.keys) == 0
}

// Add 方法用来添加缓存节点，参数为节点key，比如使用IP
func (c *ConsistentHashBanlance) Add(serviceName string,params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := params[0]
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.keys == nil{
		c.keys = make(map[string]UInt32Slice)
	}

	// 结合复制因子计算所有虚拟节点的hash值，并存入m.keys中，同时在m.hashMap中保存哈希值和key的映射

	for i := 0; i < c.replicas; i++ {
		hash := c.hash([]byte(addr+strconv.Itoa(i)))
		if php2go.InArray(hash,c.keys[serviceName]) == false{
			c.keys[serviceName] = append(c.keys[serviceName], hash)
		}
		if c.hashMap[serviceName] == nil{
			c.hashMap[serviceName] = make(map[uint32]string)
			c.hashMap[serviceName] =  map[uint32]string{hash:addr}
		}
		c.hashMap[serviceName][hash] = addr
	}
	// 对所有虚拟节点的哈希值进行排序，方便之后进行二分查找
	sort.Sort(c.keys[serviceName])
	return nil
}

// 获取本机网卡IP
func (c *ConsistentHashBanlance) getLocalIP() (ipv4 string, err error) {
	var (
		addrs []net.Addr
		addr net.Addr
		ipNet *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()	// 192.168.1.1
				return
			}
		}
	}

	err = errors.New("没有找到网卡IP")
	return
}

// Get 方法根据给定的对象获取最靠近它的那个节点
func (c *ConsistentHashBanlance) Get(key ...string) (string, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.IsEmpty() {
		return "", errors.New(" node is empty")
	}

	key1 := key[1];
	hash := c.hash([]byte(key1))

	serviceName := key[0]

	if _,ok := c.keys[serviceName];!ok{
		return "", errors.New("node is empty")
	}

	// 通过二分查找获取最优节点，第一个"服务器hash"值大于"数据hash"值的就是最优"服务器节点"
	idx := sort.Search(len(c.keys[serviceName]), func(i int) bool { return c.keys[serviceName][i] >= hash })

	// 如果查找结果 大于 服务器节点哈希数组的最大索引，表示此时该对象哈希值位于最后一个节点之后，那么放入第一个节点中
	if idx == len(c.keys[serviceName]) {
		idx = 0
	}
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.hashMap[serviceName][c.keys[serviceName][idx]], nil
}


func (c *ConsistentHashBanlance) Update() {
	//c.lock.Lock()
	//defer c.lock.Unlock()
	//log.Print("[INFO] 负载均衡器 ip_hash模式 更新负载均衡配置...... ")
	//fmt.Println("更新负载均衡配置.....")
	allServiceInfo := LoadBalanceConfig.GetLoadBalanceList()
	if allServiceInfo == nil || allServiceInfo.ServiceMap == nil{
		return
	}

	for serviceName,_  := range c.hashMap{
		if _,ok :=  allServiceInfo.ServiceMap[serviceName];!ok{
			delete(c.hashMap,serviceName)
			delete(c.keys,serviceName)
			continue
		}

		if len(allServiceInfo.ServiceMap[serviceName].Nodes) == 0{
			delete(c.hashMap,serviceName)
			delete(c.keys,serviceName)
			continue
		}

		//循环etcd中的节点信息
		var (
			tmpKeys UInt32Slice
			tmpHashMap map[uint32]string
		)
		tmpKeys = make(UInt32Slice,0)
		tmpHashMap = make(map[uint32]string,0)
		for _,etcdServiceNode := range allServiceInfo.ServiceMap[serviceName].Nodes{
			if etcdServiceNode.IP == ""{
				continue
			}
			for i := 0; i < c.replicas; i++ {
				var hash uint32
				var addr string
				if etcdServiceNode.Port == 0 {
					addr = etcdServiceNode.IP
				}else{
					addr = fmt.Sprintf("%s:%s",etcdServiceNode.IP,strconv.Itoa(etcdServiceNode.Port))
				}
				hash = c.hash([]byte(addr+strconv.Itoa(i)))
				tmpKeys = append(tmpKeys, hash)
				tmpHashMap[hash] = addr
			}
		}
		c.keys[serviceName] = tmpKeys
		c.hashMap[serviceName] = tmpHashMap
		sort.Sort(c.keys[serviceName])
	}

	for _,service := range allServiceInfo.ServiceMap{
		serviceName := service.Name
		nodes := service.Nodes
		for _,node :=range  nodes{
			if node.IP == ""{
				continue
			}
			if node.Port == 0{
				c.Add(serviceName,node.IP)
			}else{
				c.Add(serviceName,fmt.Sprintf("%s:%s",node.IP,strconv.Itoa(node.Port)))
			}
		}
	}

}

