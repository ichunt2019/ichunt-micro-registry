package load_balance

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
)

type WeightRoundRobinBalance struct {
	rss      map[string][]*WeightNode
	lock               sync.Mutex
}

type WeightNode struct {
	addr            string
	weight          int //权重值
	currentWeight   int //节点当前权重
	effectiveWeight int //有效权重
}

func (r *WeightRoundRobinBalance) Add(serviceName string,params ...string) error {
	var (
		isInsert bool
	)
	if (r.rss == nil){
		r.rss = make(map[string][]*WeightNode,0)
	}
	if len(params) != 2 {
		return errors.New("param len need 2")
	}
	weight, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil {
		return err
	}
	node := &WeightNode{addr: params[0], weight: int(weight)}
	node.effectiveWeight = node.weight
	isInsert = true
	if _,ok := r.rss[serviceName];ok{
		for k,v:=range r.rss[serviceName]{
			if v.addr == params[0]{
				isInsert = false
				r.rss[serviceName][k].weight =  int(weight)
				r.rss[serviceName][k].effectiveWeight =  int(weight)
				break
			}
		}
	}
	if isInsert{
		r.rss[serviceName] = append(r.rss[serviceName],node)
	}

	return nil
}

func (r *WeightRoundRobinBalance) Next(key string) string {
	r.lock.Lock()
	defer r.lock.Unlock()
	total := 0
	var best *WeightNode
	if r.rss == nil {
		return ""
	}
	if _,ok:=r.rss[key];!ok{
		return ""
	}

	for i := 0; i < len(r.rss[key]); i++ {
		w := r.rss[key][i]
		//step 1 统计所有有效权重之和
		total += w.effectiveWeight

		//step 2 变更节点临时权重为的节点临时权重+节点有效权重
		w.currentWeight += w.effectiveWeight

		//step 3 有效权重默认与权重相同，通讯异常时-1, 通讯成功+1，直到恢复到weight大小
		if w.effectiveWeight < w.weight {
			w.effectiveWeight++
		}
		//step 4 选择最大临时权重点节点
		if best == nil || w.currentWeight > best.currentWeight {
			best = w
		}
	}
	if best == nil {
		return ""
	}
	//step 5 变更临时权重为 临时权重-有效权重之和
	best.currentWeight -= total
	return best.addr
}

func (r *WeightRoundRobinBalance) Get(key ...string) (string, error) {
	node := r.Next(key[0])
	if node == ""{
		return "",errors.New("沒找到節點信息")
	}
	return node, nil
}

func (r *WeightRoundRobinBalance) Update() {

	//r.lock.Lock()
	//defer r.lock.Unlock()

	//log.Print("[INFO] 负载均衡器 加权轮询模式 更新负载均衡配置...... ")
	allServiceInfo := LoadBalanceConfig.GetLoadBalanceList()
	if allServiceInfo == nil || allServiceInfo.ServiceMap == nil{
		return
	}

	//删除不存在的服务节点信息
	for serviceName,_ := range r.rss{
		if _,ok :=  allServiceInfo.ServiceMap[serviceName];!ok{
			delete(r.rss,serviceName)
			continue
		}
		//循环etcd中的节点信息
		var tmpNodes []*WeightNode
		tmpNodes = make([]*WeightNode,0)
		oldTmpNodes := r.rss[serviceName]
		for _,etcdServiceNode := range allServiceInfo.ServiceMap[serviceName].Nodes{
			if etcdServiceNode.IP == ""{
				continue
			}

			if etcdServiceNode.Port == 0 {
				tmpWeightNode := &WeightNode{
					addr:etcdServiceNode.IP,
					weight:etcdServiceNode.Weight,
					currentWeight:etcdServiceNode.Weight,
					effectiveWeight:etcdServiceNode.Weight,
				}
				for _,v := range oldTmpNodes{
					if v.addr == etcdServiceNode.IP{
						tmpWeightNode.currentWeight = v.currentWeight
						tmpWeightNode.effectiveWeight = v.effectiveWeight
					}
				}
				tmpNodes = append(tmpNodes,tmpWeightNode)
			}else{
				addr := fmt.Sprintf("%s:%s",etcdServiceNode.IP,strconv.Itoa(etcdServiceNode.Port))
				tmpWeightNode := &WeightNode{
					addr:addr,
					weight:etcdServiceNode.Weight,
					currentWeight:etcdServiceNode.Weight,
					effectiveWeight:etcdServiceNode.Weight,
				}
				for _,v := range oldTmpNodes{
					if v.addr == tmpWeightNode.addr{
						tmpWeightNode.currentWeight = v.currentWeight
						tmpWeightNode.effectiveWeight = v.effectiveWeight
					}
				}
				tmpNodes = append(tmpNodes,tmpWeightNode)
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
				r.Add(serviceName,node.IP,strconv.Itoa(node.Weight))
			}else{
				r.Add(serviceName,fmt.Sprintf("%s:%s",node.IP,strconv.Itoa(node.Port)),strconv.Itoa(node.Weight))
			}
		}
	}

}

