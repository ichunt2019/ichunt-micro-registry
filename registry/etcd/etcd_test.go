package etcd

import (
	"context"
	"fmt"
	"testing"
	"time"
	"ichunt-micro/registry"
	"ichunt-micro/proxy/load_balance"
)

func TestRegister(t *testing.T) {

	//初始化注册中心  注册etcd 服务中心
	registryInst, err := registry.InitRegistry(context.TODO(), "etcd",
		registry.WithAddrs([]string{"192.168.2.232:2379"}),
		registry.WithTimeout(time.Second),
		registry.WithRegistryPath("/ichuntMicroService/"),
		registry.WithHeartBeat(5),
	)
	if err != nil {
		t.Errorf("init registry failed, err:%v", err)
		return
	}
	load_balance.Init(registryInst)

	service := &registry.Service{
		Name: "comment_service",
	}

	service.Nodes = append(service.Nodes, &registry.Node{
		IP:   "127.0.0.1",
		Port: 8801,
		Weight:2,
	},
		//&registry.Node{
		//	IP:   "127.0.0.2",
		//	Port: 8801,
		//	Weight:1,
		//},
	)
	registryInst.Register(context.TODO(), service)


	//for{
	//	go func() {
	//		loadBalance.GetService(context.TODO(), "comment_service")
	//		//loadBalance.GetService(context.TODO(), "comment_service22222")
	//	}()
	//	time.Sleep(time.Millisecond*5)
	//}

	forerver := make(chan struct{})

	<-forerver

}

func TestRegister2(t *testing.T) {
	//初始化注册中心  注册etcd 服务中心
	registryInst, err := registry.InitRegistry(context.TODO(), "etcd",
		registry.WithAddrs([]string{"192.168.2.232:2379"}),
		registry.WithTimeout(time.Second),
		registry.WithRegistryPath("/ichuntMicroService/"),
		registry.WithHeartBeat(5),
	)
	if err != nil {
		t.Errorf("init registry failed, err:%v", err)
		return
	}

	load_balance.Init(registryInst)

	service := &registry.Service{
		Name: "comment_service",
	}

	service.Nodes = append(service.Nodes, &registry.Node{
		IP:   "127.0.0.3",
		Port: 8801,
		Weight:1,
	},
		//&registry.Node{
		//	IP:   "127.0.0.4",
		//	Port: 8801,
		//	Weight:2,
		//},
	)
	registryInst.Register(context.TODO(), service)

	forerver := make(chan struct{})

	<-forerver

}

func TestGetService(t *testing.T){

	//初始化注册中心  注册etcd 服务中心
	registryInst, err := registry.InitRegistry(context.TODO(), "etcd",
		registry.WithAddrs([]string{"192.168.2.232:2379"}),
		registry.WithTimeout(time.Second),
		registry.WithPasswrod(""),
		registry.WithRegistryPath("/ichuntMicroService/"),
		registry.WithHeartBeat(5),
	)
	if err != nil {
		fmt.Printf("init registry failed, err:%v", err)
		return
	}

	loadBalance := load_balance.Init(registryInst)


	for{
		go func() {
			loadBalance.GetService(context.TODO(), "comment_service")
			//loadBalance.GetService(context.TODO(), "comment_service22222")
		}()
		time.Sleep(time.Second*1)
		//time.Sleep(time.Millisecond*100)
	}

}
