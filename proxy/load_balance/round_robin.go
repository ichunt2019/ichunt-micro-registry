package load_balance

import (
	"errors"
	"fmt"
	"sync"
	"github.com/syyongx/php2go"
	"strconv"
)

type RoundRobinBalance struct {
	curIndex map[string]int
	rss     map[string][]string
	lock    sync.Mutex
}

func (r *RoundRobinBalance) Add(serviceName string,params ...string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := params[0]
	if (r.rss == nil){
		r.rss = make(map[string][]string,0)
	}

	if (r.curIndex == nil){
		r.curIndex = make(map[string]int,0)
	}

	if php2go.InArray(addr,r.rss[serviceName]) == false{
		r.rss[serviceName] = append(r.rss[serviceName],addr)
		if _,ok:=r.curIndex[serviceName] ;!ok{
			r.curIndex[serviceName] = 0
		}
		//r.curIndex[serviceName] = 0
	}

	return nil
}

func (r *RoundRobinBalance) Next(serviceName string) string {
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.rss == nil || len(r.rss) == 0 {
		return ""
	}

	if _,ok := r.rss[serviceName]; !ok{
		return ""
	}


	if _,ok :=r.curIndex[serviceName];!ok{
		return ""
	}
	currentInt := r.curIndex[serviceName]


	lens := len(r.rss[serviceName]) //节点的个数

	if (r.curIndex == nil){
		r.curIndex = make(map[string]int,0)
		r.curIndex[serviceName] = 0
	}

	if currentInt >= lens {
		r.curIndex[serviceName] = 0
		currentInt = 0
	}

	curAddr := r.rss[serviceName][currentInt]

	r.curIndex[serviceName] = (currentInt + 1) % lens
	return curAddr
}

func (r *RoundRobinBalance) Get(key ...string) (string, error) {
	node := r.Next(key[0])
	if node == ""{
		return "",errors.New("沒找到節點信息")
	}
	return node, nil
}

/*
更新负载均衡器重缓存服务的节点信息
 */
func (r *RoundRobinBalance) Update() {

	//log.Print("[INFO] 负载均衡器 轮询模式 更新负载均衡配置...... ")
	//fmt.Println("更新负载均衡配置.....")
	allServiceInfo := LoadBalanceConfig.GetLoadBalanceList()
	if allServiceInfo == nil || allServiceInfo.ServiceMap == nil{
		return
	}

	//删除不存在的服务节点信息
	for serviceName,_ := range r.rss{
		if _,ok :=  allServiceInfo.ServiceMap[serviceName];!ok{
			delete(r.rss,serviceName)
			delete(r.curIndex,serviceName)
			continue
		}
		//循环etcd中的节点信息
		var tmpNodes []string
		for _,etcdServiceNode := range allServiceInfo.ServiceMap[serviceName].Nodes{
			if etcdServiceNode.IP == ""{
				continue
			}
			if etcdServiceNode.Port == 0 {
				tmpNodes = append(tmpNodes,etcdServiceNode.IP)
			}else{
				tmpNodes = append(tmpNodes,fmt.Sprintf("%s:%s",etcdServiceNode.IP,strconv.Itoa(etcdServiceNode.Port)))
			}
		}
		r.rss[serviceName] = tmpNodes
		if r.curIndex[serviceName] > len(r.rss[serviceName]){
			r.curIndex[serviceName] = 0
		}
	}

	for _,service := range allServiceInfo.ServiceMap{
		serviceName := service.Name
		nodes := service.Nodes
		for _,node :=range  nodes{
			if node.IP == ""{
				continue
			}
			if node.Port == 0{
				r.Add(serviceName,node.IP)
			}else{
				r.Add(serviceName,fmt.Sprintf("%s:%s",node.IP,strconv.Itoa(node.Port)))
			}
		}
	}

}
