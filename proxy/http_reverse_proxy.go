package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/syyongx/php2go"
	"ichunt-micro/dao"
	"ichunt-micro/golang_common/lib"
	"ichunt-micro/golang_common/log"
	"ichunt-micro/middleware"
	"ichunt-micro/proxy/load_balance"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (

	transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, //连接超时
			KeepAlive: 30 * time.Second, //长连接超时时间
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,              //最大空闲连接
		IdleConnTimeout:       60 * time.Second, //空闲超时时间
		TLSHandshakeTimeout:   10 * time.Second, //tls握手超时时间
		ResponseHeaderTimeout: 30*time.Second,
		ExpectContinueTimeout: 1 * time.Second,  //100-continue状态码超时时间
	}
)

func NewMultipleHostsReverseProxy(c *gin.Context) (*httputil.ReverseProxy, error) {
	var (
		ServiceName string
		target *url.URL
		err error
		nextAddr string
	)
	//请求协调者
	service,exists := c.Get("service")
	if !exists {
		err := errors.New("get service  fail")
		log.Error("%s", err)
		return nil,err
	}
	service_,ok := service.(*dao.ServiceDetail)
	if !ok{
		err := errors.New("get service  fail")
		log.Error("%s", err)
		return nil,err
	}
	//数据库中的rule字段  前缀匹配
	ServiceName = service_.Info.ServiceName
	ServiceName = php2go.Trim(ServiceName,"/")
	ServiceName = lib.CompressStr(ServiceName)
	//fmt.Println("service_.HTTPRule.NeedDirectForward",service_.HTTPRule.NeedDirectForward)
	//fmt.Println("service_.HTTPRule.DirectForwardUrl",service_.HTTPRule.DirectForwardUrl)
	if service_.HTTPRule.NeedDirectForward == 0 {
		//服务的 负载均衡方式 轮询方式 0=random 1=round-robin 2=weight_round-robin 3=ip_hash
		round_type := service_.LoadBalance.RoundType
		nextAddr, err = load_balance.LoadBalanceConfig.GetService(context.TODO(),round_type, ServiceName,lib.ClientIP(c.Request))
		if err != nil || nextAddr == "" {
			err = errors.New(fmt.Sprintf("从etcd中获取服务 %s  失败",ServiceName))
			log.Error("%s", err)
			return nil,err
		}
	}else{
		//直接转发url
		nextAddr = service_.HTTPRule.DirectForwardUrl
	}

	director := func(req *http.Request) {
		if strings.HasPrefix(nextAddr,"http://") || strings.HasPrefix(nextAddr,"https://") {
			target, err = url.Parse(nextAddr)
		}else{
			target, err = url.Parse("http://"+nextAddr)
		}
		if err != nil {
			log.Error("func NewMultipleHostsReverseProxy 匿名函数director  url.Parse地址解析失败 失败 %s",err)
		}
		//fmt.Println("target",target)
		//http://192.168.1.234:2004
		targetQuery := target.RawQuery
		//http
		req.URL.Scheme = target.Scheme
		//192.168.1.234:2004
		req.URL.Host = target.Host
		req.Host = target.Host

		///comment_service/test
		//fmt.Println("target.Path",target.Path)
		//fmt.Println("req.URL.Path",req.URL.Path)
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		//req.URL.Path = php2go.StrReplace(rule,"",req.URL.Path,-1)
		log.Info("target %s",target)
		log.Info("targetQuery %s",targetQuery)
		log.Info("req.URL.Scheme %s",req.URL.Scheme)
		log.Info("req.URL.Host %s",req.URL.Host)
		log.Info("req.URL.Path %s",req.URL.Path)

		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		//fmt.Println("req.Header",req.Header)
		//fmt.Println("req.Method",req.Method)
		//fmt.Println("req.Body",req.Body)
		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}

		//只在第一代理中设置此header头
		req.Header.Set("X-Real-Ip", req.RemoteAddr)
	}

	//更改内容
	modifyFunc := func(resp *http.Response) error {
		if strings.Contains(resp.Header.Get("Connection"), "Upgrade") {
			return nil
		}
		var payload []byte
		var readErr error
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			payload, readErr = ioutil.ReadAll(gr)
			resp.Header.Del("Content-Encoding")
		} else {
			payload, readErr = ioutil.ReadAll(resp.Body)
		}
		if readErr != nil {
			return readErr
		}
		//异常请求时设置StatusCode
		if resp.StatusCode != 200 {
			payload = []byte("StatusCode error:" + string(payload))
		}
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		resp.ContentLength = int64(len(payload))
		resp.Header.Set("Content-Length", strconv.FormatInt(int64(len(payload)), 10))
		return nil
	}



	//错误回调 ：关闭real_server时测试，错误回调
	//范围：transport.RoundTrip发生的错误、以及ModifyResponse发生的错误
	errFunc := func(w http.ResponseWriter, r *http.Request, err error) {
		//todo 如果是权重的负载则调整临时权重
		//http.Error(w, "ErrorHandler error:"+err.Error(), 500)
		middleware.ResponseError(c,500,err)
	}
	//fmt.Println(director)
	return &httputil.ReverseProxy{Director: director, Transport:transport, ModifyResponse: modifyFunc, ErrorHandler: errFunc},nil
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
