package config

import (
	"time"
	"context"
	"fmt"
	"github.com/ichunt2019/ichunt-micro-registry/registry"
	_ "github.com/ichunt2019/ichunt-micro-registry/registry/etcd"

	)

func Register(microServieName string,etcdConfig registry.EtcdConfig,nodes []*registry.Node){
	registryInst, err := registry.InitRegistry(context.TODO(), "etcd",
		registry.WithAddrs(etcdConfig.Address),
		registry.WithTimeout(time.Second),
		registry.WithPasswrod(etcdConfig.Username),
		registry.WithPasswrod(etcdConfig.Password),
		registry.WithRegistryPath(etcdConfig.Path),
		registry.WithHeartBeat(5),
	)

	if err != nil {
		fmt.Printf("init registry failed, err:%v", err)
		return
	}

	//load_balance.Init(registryInst)

	service := &registry.Service{
		Name: microServieName,
	}

	service.Nodes = append(service.Nodes,
		nodes...,
	)
	registryInst.Register(context.TODO(), service)
}
