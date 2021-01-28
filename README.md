# ichunt-micro-registry
## golang微服务网关注册

`import引入`

"github.com/ichunt2019/ichunt-micro-registry/registry"
"github.com/ichunt2019/ichunt-micro-registry/config"
_ "github.com/ichunt2019/ichunt-micro-registry/registry/etcd"



```
func main(){

    

}
```




```
nodes := []*registry.Node{
		{
			IP:   "192.168.2.246",
			Port: 2004,
			Weight:2,
		},
	}

	etcdConfig := registry.EtcdConfig{
		Address:  []string{"192.168.2.232:2379"},
		Username: "",
		Password: "",
		Path:"/ichuntMicroService/",
	}
	config.Register("sku_server",etcdConfig,nodes)
```
