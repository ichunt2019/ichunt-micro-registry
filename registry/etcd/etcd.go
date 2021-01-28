package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	//"github.com/ichunt2019/ichunt-micro-registry/proxy/load_balance"
	"github.com/ichunt2019/ichunt-micro-registry/registry"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MaxServiceNum          = 50
	MaxSyncServiceInterval = time.Second * 30
)

//etcd 注册插件
type EtcdRegistry struct {
	options   *registry.Options//etcd配置信息
	client    *clientv3.Client
	serviceCh chan *registry.Service//服务注册 -》 服务信息   节点信息 servicename node

	value              atomic.Value //缓存已经注册的服务节点信息
	lock               sync.Mutex
	registryServiceMap map[string]*RegisterService//需要注册到etcd中的服务节点信息
}


//存放所有服务信息 存入atomic.Value 为了防止并发
//type AllServiceInfo struct {
//	serviceMap map[string]*registry.Service  //节点信息 servicename node
//}

type RegisterService struct {
	id          clientv3.LeaseID
	service     *registry.Service //节点信息 servicename node
	registered  bool
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse   //etcd返回的续租的信息  false表示续租失败  续租应答
}

var (
	//实例化etcd服务
	etcdRegistry *EtcdRegistry = &EtcdRegistry{
		serviceCh:          make(chan *registry.Service, MaxServiceNum),
		registryServiceMap: make(map[string]*RegisterService, MaxServiceNum),
	}
	once sync.Once
)

//导入包立即执行函数
func init() {
	allServiceInfo := &registry.AllServiceInfo{
		ServiceMap: make(map[string]*registry.Service, MaxServiceNum),
	}
	//atomic.Value  原子操作 为了防止并发
	etcdRegistry.value.Store(allServiceInfo)
	//注册etcd插件
	//map["etcd"] = etcdRegistry
	registry.RegisterPlugin(etcdRegistry)
	go etcdRegistry.run()
}

//插件的名字
func (e *EtcdRegistry) Name() string {
	return "etcd"
}

//初始化
func (e *EtcdRegistry) Init(ctx context.Context, opts ...registry.Option) (err error) {

	e.options = &registry.Options{}
	for _, opt := range opts {
		opt(e.options)
	}

	clientConfig := clientv3.Config{
		Endpoints:   e.options.Addrs,
		DialTimeout: e.options.Timeout,
		Password:	 e.options.Password,
		Username:    e.options.Username,
	}

	e.client, err = clientv3.New(clientConfig)
	if err != nil {
		panic(fmt.Sprintf("初始化etcd失败, err:%v", err))
		return
	}
	//下面代码 是因为当etcd ip链接不同 不报错导致超时 阻塞的办法
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	_, err =e.client.Status(timeoutCtx, clientConfig.Endpoints[0])
	if err != nil {
		panic(fmt.Sprintf("etcd启动失败 %s",err))
		return
	}

	return
}

//服务注册 把服务名称 节点信息放入通道serviceCh
func (e *EtcdRegistry) Register(ctx context.Context, service *registry.Service) (err error) {

	select {
	case e.serviceCh <- service:
	default:
		err = fmt.Errorf("register chan is full")
		return
	}
	return
}

//服务反注册
func (e *EtcdRegistry) Unregister(ctx context.Context, service *registry.Service) (err error) {
	return
}

func (e *EtcdRegistry) run() {

	ticker := time.NewTicker(MaxSyncServiceInterval)
	for {
		select {
		//读取注册进来的节点信息（服务名 ip+端口）
		//手动加载过来的服务
		case service := <-e.serviceCh:
			//读取已经注册的服务
			registryService, ok := e.registryServiceMap[service.Name]
			if ok {
				//更新节点信息
				for _, node := range service.Nodes {
					registryService.service.Nodes = append(registryService.service.Nodes, node)
				}
				registryService.registered = false
				break
			}
			//插入节点信息
			registryService = &RegisterService{
				service: service,
			}
			//需要注册到etcd中的服务信息
			e.registryServiceMap[service.Name] = registryService
		case <-ticker.C:
			//定时(10秒钟) 更新缓存中的服务信息
			//e.value  = AllServiceInfo
			e.syncServiceFromEtcd()
		default:
			//注册服务 并 续约(把手动传过的需要注册的服务  put到etcd中 续租 )
			//续租应答
			e.registerOrKeepAlive()
			time.Sleep(time.Millisecond * 500)
		}
	}
}


/*
服务注册
服务续租
 */
func (e *EtcdRegistry) registerOrKeepAlive() {
	//循环注册的节点信息
	for _, registryService := range e.registryServiceMap {
		//如果是存活的节点 节点续期
		if registryService.registered {
			//处理续租应答  如果续租失败 registered = false
			e.keepAlive(registryService)
			continue
		}
		//服务注册    并永久续约
		//设置registered = true
		e.registerService(registryService)
	}
}


/*
处理续租应答的协程
如果续租失败 registered = false
 */
func (e *EtcdRegistry) keepAlive(registryService *RegisterService) {

	select {
	case resp := <-registryService.keepAliveCh:
		if resp == nil {
			//租约失效 去除节点
			registryService.registered = false
			return
		}
	}
	return
}

func (e *EtcdRegistry) registerService(registryService *RegisterService) (err error) {

	resp, err := e.client.Grant(context.TODO(), e.options.HeartBeat)
	if err != nil {
		return
	}

	registryService.id = resp.ID
	for _, node := range registryService.service.Nodes {

		tmp := &registry.Service{
			Name: registryService.service.Name,
			Nodes: []*registry.Node{
				node,
			},
		}

		data, err := json.Marshal(tmp)
		if err != nil {
			continue
		}

		key := e.serviceNodePath(tmp)
		fmt.Printf("register key:%s\n", key)
		_, err = e.client.Put(context.TODO(), key, string(data), clientv3.WithLease(resp.ID))
		if err != nil {
			continue
		}

		// 自动续租
		//<-ch   <-ch==nil 租约失效
		ch, err := e.client.KeepAlive(context.TODO(), resp.ID)
		if err != nil {
			continue
		}

		registryService.keepAliveCh = ch //续租
		registryService.registered = true
	}

	return
}

func (e *EtcdRegistry) serviceNodePath(service *registry.Service) string {

	nodeIP := fmt.Sprintf("%s:%d", service.Nodes[0].IP, service.Nodes[0].Port)
	return path.Join(e.options.RegistryPath, service.Name, nodeIP)
}

func (e *EtcdRegistry) servicePath(name string) string {
	return path.Join(e.options.RegistryPath, name)
}

func (e *EtcdRegistry) getServiceFromCache(ctx context.Context,
	name string) (service *registry.Service, ok bool) {

	allServiceInfo := e.value.Load().(*registry.AllServiceInfo)
	//一般情况下，都会从缓存中读取
	service, ok = allServiceInfo.ServiceMap[name]
	return
}

func (e *EtcdRegistry) GetService(ctx context.Context, name string) (service *registry.Service, err error) {

	//如果缓存中没有这个service，则从etcd中读取
	e.lock.Lock()
	defer e.lock.Unlock()
	service, ok := e.getServiceFromCache(ctx, name)
	if ok {
		//fmt.Println("从缓存中获取...")
		return
	}
	//从etcd中读取指定服务名字的服务信息
	key := e.servicePath(name)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return
	}

	service = &registry.Service{
		Name: name,
	}

	for _, kv := range resp.Kvs {
		value := kv.Value
		var tmpService registry.Service
		err = json.Unmarshal(value, &tmpService)
		if err != nil {
			return
		}

		for _, node := range tmpService.Nodes {
			if tmpService.Name != name{
				continue
			}
			service.Nodes = append(service.Nodes, node)
		}
	}
	//从内存中读取所有的服务列表
	allServiceInfoOld := e.value.Load().(*registry.AllServiceInfo)
	//声明新的缓存服务
	var allServiceInfoNew = &registry.AllServiceInfo{
		ServiceMap: make(map[string]*registry.Service, MaxServiceNum),
	}
	//copy旧的服务列表信息到新的变量中
	for key, val := range allServiceInfoOld.ServiceMap {
		allServiceInfoNew.ServiceMap[key] = val
	}
	//更新当前需要获取的服务列表
	allServiceInfoNew.ServiceMap[name] = service
	e.value.Store(allServiceInfoNew)

	//load_balance.LoadBalanceConfig.UpdateConf(allServiceInfoNew)

	return
}

/*
读取缓存从etcd中拉取服务节点信息 并更新
 */
func (e *EtcdRegistry) syncServiceFromEtcd() {
	//fmt.Println("更新缓存的服务列表...")
	var allServiceInfoNew = &registry.AllServiceInfo{
		ServiceMap: make(map[string]*registry.Service, MaxServiceNum),
	}

	ctx := context.TODO()
	allServiceInfo := e.value.Load().(*registry.AllServiceInfo)
	//对于缓存的每一个服务，都需要从etcd中进行更新
	for _, service := range allServiceInfo.ServiceMap {
		key := e.servicePath(service.Name)
		resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
		if err != nil {
			allServiceInfoNew.ServiceMap[service.Name] = service
			continue
		}

		serviceNew := &registry.Service{
			Name: service.Name,
		}

		for _, kv := range resp.Kvs {
			value := kv.Value
			var tmpService registry.Service
			err = json.Unmarshal(value, &tmpService)
			if err != nil {
				fmt.Printf("unmarshal failed, err:%v value:%s", err, string(value))
				return
			}
			if tmpService.Name != service.Name{
				continue
			}
			for _, node := range tmpService.Nodes {
				serviceNew.Nodes = append(serviceNew.Nodes, node)
			}
		}
		allServiceInfoNew.ServiceMap[serviceNew.Name] = serviceNew
	}

	e.value.Store(allServiceInfoNew)
	//load_balance.LoadBalanceConfig.UpdateConf(allServiceInfoNew)
}
