# ichunt-micro-registry
## golang微服务网关注册

`import引入`

"github.com/ichunt2019/ichunt-micro-registry/registry"
_ "github.com/ichunt2019/ichunt-micro-registry/registry/etcd"



```
func main(){

    //服务注册
    register()

}
```




```
func register(){
	registryInst, err := registry.InitRegistry(context.TODO(), "etcd",
		registry.WithAddrs([]string{"192.168.2.232:2379"}),//服务器etcd ip地址
		registry.WithTimeout(time.Second),
		registry.WithPasswrod(""),//etcd密码
		registry.WithRegistryPath("/ichuntMicroService/"),//服务前缀 固定
		registry.WithHeartBeat(5),
	)

	if err != nil {
		fmt.Printf("init registry failed, err:%v", err)
		return
	}

	service := &registry.Service{
		Name: "comment_service",//注册服务的名称
	}

	//服务节点信息 ip  端口  权重
	service.Nodes = append(service.Nodes,
		&registry.Node{
			IP:   "192.168.2.246",
			Port: 2004,
			Weight:50,//参考是50
		},
	)
	registryInst.Register(context.TODO(), service)
}
```
