package load_balance

import (
	"context"
	"fmt"
	"github.com/ichunt2019/ichunt-micro-registry/registry"
	"sync"
	"sync/atomic"
)

type Observer interface {
	Update()
}


type LoadBalanceEtcdConf struct {
	observers    []Observer
	value              atomic.Value //缓存已经注册的服务节点信息
	registry registry.Registry
}


var(
	once sync.Once
	//实例化对象
	LoadBalanceConfig *LoadBalanceEtcdConf
)

func Init(reg registry.Registry) *LoadBalanceEtcdConf{
	once.Do(func() {
		LoadBalanceConfig  = &LoadBalanceEtcdConf{}
		rb0 := LoadBanlanceFactory(LbRandom)
		rb1 := LoadBanlanceFactory(LbRoundRobin)
		rb2 := LoadBanlanceFactory(LbWeightRoundRobin)
		rb3 := LoadBanlanceFactory(LbConsistentHash)
		//添加负载均衡器到配置中
		LoadBalanceConfig.Attach(rb0)
		LoadBalanceConfig.Attach(rb1)
		LoadBalanceConfig.Attach(rb2)
		LoadBalanceConfig.Attach(rb3)
		LoadBalanceConfig.registry = reg
		allServiceInfo := &registry.AllServiceInfo{
			ServiceMap: make(map[string]*registry.Service),
		}
		//atomic.Value  原子操作 为了防止并发
		LoadBalanceConfig.value.Store(allServiceInfo)
	})
	return LoadBalanceConfig
}

func (s *LoadBalanceEtcdConf) Attach(o Observer) {
	s.observers = append(s.observers, o)
}

//更新配置时，通知监听者也更新
func (s *LoadBalanceEtcdConf) UpdateConf(serviceInfo *registry.AllServiceInfo) {
	if len(serviceInfo.ServiceMap) == 0 {
		return
	}
	s.value.Store(serviceInfo)
	for _, obs := range s.observers {
		obs.Update()
	}
}

func (s *LoadBalanceEtcdConf) GetLoadBalanceList() *registry.AllServiceInfo{
	return s.value.Load().(*registry.AllServiceInfo)
}

func (s *LoadBalanceEtcdConf) GetLoadBalance(round_type int) LoadBalance{
	return s.observers[round_type].(LoadBalance)
}

func (s *LoadBalanceEtcdConf) GetRegistry() registry.Registry{
	return s.registry
}

func (s *LoadBalanceEtcdConf) GetService(ctx context.Context,round_type int,name string,ip string)(node string,err error){
	regService := s.GetRegistry()
	_, err = regService.GetService(context.TODO(),name)
	if err != nil {
		fmt.Printf("get service failed, err:%v", err)
		return
	}
	//轮询方式 0=random 1=round-robin 2=weight_round-robin 3=ip_hash
	if round_type > 3{
		round_type=2
	}
	node ,err =s.GetLoadBalance(round_type).Get(name,ip)
	return node,err
}






