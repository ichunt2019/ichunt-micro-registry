package main

import (
	"fmt"
	"github.com/ichunt2019/ichunt-micro-registry/registry"
	"github.com/ichunt2019/ichunt-micro-registry/config"
	_ "github.com/ichunt2019/ichunt-micro-registry/registry/etcd"
	_ "github.com/imroc/req"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	//rs1 := &RealServer{Addr: "192.168.2.232:2003"}
	//rs1.Run()

	//rs2 := &RealServer{Addr: "192.168.2.232:2004"}
	//rs2 := &RealServer{Addr: "192.168.1.234:2004"}
	//rs2 := &RealServer{Addr: "192.168.1.237:2004"}
	rs2 := &RealServer{Addr: "192.168.2.246:2004"}
	rs2.Run()

	//服务注册
	//register()

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


	//监听关闭信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type RealServer struct {
	Addr string
}


func (r *RealServer) Run() {
	log.Println("Starting httpserver at " + r.Addr)
	mux := http.NewServeMux()
	mux.HandleFunc("/test", r.HelloHandler)
	mux.HandleFunc("/base/error", r.ErrorHandler)
	server := &http.Server{
		Addr:         r.Addr,
		WriteTimeout: time.Second * 3,
		Handler:      mux,
	}
	go func() {
		log.Fatal(server.ListenAndServe())
	}()
}

func (r *RealServer) HelloHandler(w http.ResponseWriter, req *http.Request) {
	//127.0.0.1:8008/abc?sdsdsa=11
	//r.Addr=127.0.0.1:8008
	//req.URL.Path=/abc
	//time.Sleep(time.Second)
	fmt.Println("host:",req.Host)
	fmt.Println("header:",req.Header)
	fmt.Println("cookie:",req.Cookies())
	fmt.Println(req.ParseForm())
	fmt.Println("post params: ",req.PostForm)
	fmt.Println("url :",req.URL)
	fmt.Println("url rawpath :",req.URL.RawPath)
	fmt.Println("query :",req.URL.Query())

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("read body err, %v\n", err)
		return
	}
	println("json:", string(body))

	upath := fmt.Sprintf("http://%s%s\n", r.Addr, req.URL.Path)
	realIP := fmt.Sprintf("RemoteAddr=%s,X-Forwarded-For=%v,X-Real-Ip=%v\n", req.RemoteAddr, req.Header.Get("X-Forwarded-For"),
		req.Header.Get("X-Real-Ip"))
	io.WriteString(w, upath)
	io.WriteString(w, realIP)
}

func (r *RealServer) ErrorHandler(w http.ResponseWriter, req *http.Request) {
	upath := "error handler"
	w.WriteHeader(500)
	io.WriteString(w, upath)
}