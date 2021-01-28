package registry

// 服务抽象
type Service struct {
	Name  string  `json:"name"`
	Nodes []*Node `json:"nodes"`
}

// 服务节点的抽象
type Node struct {
	Id     string `json:"id"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Weight int    `json:"weight"`
}


type AllServiceInfo struct {
	ServiceMap map[string]*Service  //节点信息 servicename node
}


type EtcdConfig struct {
	Address []string
	Username     string
	Password     string
	Path     string
}