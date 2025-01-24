package iterator

import (
	"testing"

	"github.com/ipfans/authgate/proxy"
	"github.com/stretchr/testify/assert"
)

func TestNewRoundRobin(t *testing.T) {
	// 创建测试用的代理实例
	proxy1, _ := proxy.New("http://proxy1:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy2, _ := proxy.New("http://proxy2:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy3, _ := proxy.New("http://proxy3:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})

	proxies := []*proxy.Proxy{proxy1, proxy2, proxy3}

	// 测试创建 RoundRobin 迭代器
	rr := NewRoundRobin(proxies...)
	assert.NotNil(t, rr)

	// 验证类型转换
	_, ok := rr.(*RoundRobin)
	assert.True(t, ok)
}

func TestRoundRobin_Next(t *testing.T) {
	// 创建测试用的代理实例
	proxy1, _ := proxy.New("http://proxy1:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy2, _ := proxy.New("http://proxy2:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy3, _ := proxy.New("http://proxy3:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})

	proxies := []*proxy.Proxy{proxy1, proxy2, proxy3}
	rr := NewRoundRobin(proxies...).(*RoundRobin)

	// 测试循环行为
	expectedSequence := []*proxy.Proxy{proxy1, proxy2, proxy3, proxy1}

	for _, expected := range expectedSequence {
		actual, err := rr.Next()
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	}
}

func TestRoundRobin_Next_Empty(t *testing.T) {
	// 测试空代理列表的情况
	rr := NewRoundRobin()

	proxy, err := rr.Next()
	assert.Error(t, err)
	assert.Nil(t, proxy)
}

func TestRoundRobin_Next_Concurrent(t *testing.T) {
	// 创建测试用的代理实例
	proxy1, _ := proxy.New("http://proxy1:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy2, _ := proxy.New("http://proxy2:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy3, _ := proxy.New("http://proxy3:8080", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})

	proxies := []*proxy.Proxy{proxy1, proxy2, proxy3}

	rr := NewRoundRobin(proxies...)

	// 并发测试
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				proxy, err := rr.Next()
				assert.NoError(t, err)
				assert.NotNil(t, proxy)
				assert.Contains(t, proxies, proxy)
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
