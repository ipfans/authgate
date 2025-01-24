package proxy

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/reverseproxy"
	"github.com/ipfans/authgate/utils/defaults"
)

type Proxy struct {
	reverseproxy.ReverseProxy
	clientConfig []config.ClientOption
	healthCheck  HealthCheck
	mu           sync.RWMutex
	healthState  bool
	connNum      int
}

type ClientConfig struct {
	DialTimeout         time.Duration
	MaxConnsPerHost     int
	MaxIdleConnDuration time.Duration
	KeepAlive           bool
	ReadTimeout         time.Duration
	ResponseBodyStream  bool
	WriteTimeout        time.Duration
}

type HealthCheck struct {
	Enabled          bool     `koanf:"enabled"`            // 是否启用健康检查
	Host             string   `koanf:"host"`               // 检查主机名
	Method           string   `koanf:"method"`             // 检查方法, 默认 GET
	Path             string   `koanf:"path"`               // 检查路径
	Interval         int      `koanf:"interval"`           // 检查间隔, 单位秒
	Timeout          int      `koanf:"timeout"`            // 超时时间, 单位秒
	AllowStatusCodes []string `koanf:"allow_status_codes"` // 允许的状态码, 默认 2xx, 3xx
}

func New(upstream string, healthCheck HealthCheck, clientConfig ClientConfig) (*Proxy, error) {
	p := &Proxy{}
	opts := []config.ClientOption{
		client.WithDialTimeout(defaults.Get(clientConfig.DialTimeout, consts.DefaultDialTimeout)),
		client.WithWriteTimeout(defaults.Get(clientConfig.WriteTimeout, time.Minute)),
		client.WithClientReadTimeout(defaults.Get(clientConfig.ReadTimeout, time.Minute)),
		client.WithMaxIdleConnDuration(defaults.Get(clientConfig.MaxIdleConnDuration, consts.DefaultMaxIdleConnDuration)),
		client.WithMaxConnsPerHost(defaults.Get(clientConfig.MaxConnsPerHost, consts.DefaultMaxConnsPerHost)),
		client.WithKeepAlive(clientConfig.KeepAlive),
		client.WithResponseBodyStream(clientConfig.ResponseBodyStream),
		client.WithConnStateObserve(func(state config.HostClientState) {
			p.mu.Lock()
			p.connNum = state.ConnPoolState().TotalConnNum
			p.mu.Unlock()
		}, time.Second/10),
	}

	healthCheck.Method = defaults.Get(healthCheck.Method, "GET")
	healthCheck.Path = defaults.Get(healthCheck.Path, "/")
	healthCheck.Interval = defaults.Get(healthCheck.Interval, 10)
	healthCheck.Timeout = defaults.Get(healthCheck.Timeout, 5)

	rp, err := reverseproxy.NewSingleHostReverseProxy(upstream, opts...)
	if err != nil {
		return nil, err
	}

	p.ReverseProxy = *rp
	p.clientConfig = opts
	p.healthCheck = healthCheck
	p.startHealthCheck()
	return p, nil
}

func (p *Proxy) healthChecking() bool {
	// 获取客户端实例
	client, err := client.NewClient(p.clientConfig...)
	if err != nil {
		return false
	}

	// 构造健康检查请求
	req := &protocol.Request{}
	req.Header.SetMethod(p.healthCheck.Method)
	path := p.healthCheck.Path
	req.SetHost(p.healthCheck.Host)
	req.SetRequestURI(path)

	// 设置超时时间
	timeout := time.Duration(p.healthCheck.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 发送请求
	resp := &protocol.Response{}
	err = client.Do(ctx, req, resp)
	if err != nil {
		return false
	}

	// 检查响应状态码
	if len(p.healthCheck.AllowStatusCodes) == 0 {
		return resp.StatusCode() >= 200 && resp.StatusCode() < 400
	}

	// 检查响应状态码
	statusCode := resp.StatusCode()
	for _, code := range p.healthCheck.AllowStatusCodes {
		// 解析状态码范围,如 2xx 表示 200-299
		if len(code) == 3 && code[1:] == "xx" {
			base := int((code[0] - '0') * 100)
			if statusCode >= base && statusCode < base+100 {
				return true
			}
		} else {
			// 检查单个状态码
			if code == strconv.Itoa(statusCode) {
				return true
			}
		}
	}

	return false
}

func (p *Proxy) startHealthCheck() {
	if !p.healthCheck.Enabled {
		p.mu.Lock()
		p.healthState = true
		p.mu.Unlock()
		return // 如果健康检查未启用，直接返回
	}

	go func() {
		ticker := time.NewTicker(time.Duration(p.healthCheck.Interval) * time.Second)
		defer ticker.Stop()

		for {
			state := p.healthChecking()
			p.mu.Lock()
			p.healthState = state
			p.mu.Unlock()

			<-ticker.C
		}
	}()
}

// IsAvailable 获取代理的可用性
func (p *Proxy) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthState
}

// GetLoad 获取代理的负载
func (p *Proxy) GetLoad() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connNum
}

// SetAvailable 设置代理的可用性，测试用功能不应实际使用
func (p *Proxy) SetAvailable(available bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthState = available
}

// SetLoad 设置代理的负载，测试用功能不应实际使用
func (p *Proxy) SetLoad(load int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connNum = load
}
