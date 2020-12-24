package load_balance

import (
	"errors"
	"fmt"
	"github.com/syyongx/php2go"
	"math/rand"
	"strconv"
	"sync"
)

type RandomBalance struct {
	curIndex map[string]int
	rss      map[string][]string
	lock               sync.Mutex
}

func (r *RandomBalance) Add(serviceName string,params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := params[0]
	if (r.rss == nil){
		r.rss = make(map[string][]string,0)
	}
	if(r.curIndex == nil){
		r.curIndex = make(map[string]int,0)
	}
	if php2go.InArray(addr,r.rss[serviceName]) == false{
		r.rss[serviceName] = append(r.rss[serviceName],addr)
		r.curIndex[serviceName] = 0
	}

	return nil
}

func (r *RandomBalance) Next(serviceName string) string {
	r.lock.Lock()
	defer r.lock.Unlock()
	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	if r.rss == nil || len(r.rss) == 0 {
		return ""
	}

	serviceList,ok := r.rss[serviceName]
	if !ok{
		return ""
	}
	randNum := rand.Intn(len(serviceList))
	return serviceList[randNum]
}

func (r *RandomBalance) Get(key ...string) (string, error) {
	node := r.Next(key[0])
	if node == ""{
		return "",errors.New("沒找到節點信息")
	}
	return node, nil
}



func (r *RandomBalance) Update() {
	//r.lock.Lock()
	//defer r.lock.Unlock()
	//log.Print("[INFO] 负载均衡器 随机模式 更新负载均衡配置...... ")
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
