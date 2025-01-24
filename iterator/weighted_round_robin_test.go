package iterator

import (
	"testing"

	"github.com/ipfans/authgate/proxy"
	"github.com/stretchr/testify/assert"
)

func TestNewWeightedRoundRobin(t *testing.T) {
	// 创建测试代理
	proxy1 := &proxy.Proxy{}
	proxy2 := &proxy.Proxy{}

	// 创建权重映射
	proxies := map[*proxy.Proxy]int32{
		proxy1: 2,
		proxy2: 1,
	}

	// 初始化迭代器
	wrr := NewWeightedRoundRobin(proxies)

	// 验证类型
	_, ok := wrr.(*WeightedRoundRobin)
	assert.True(t, ok, "应该返回 WeightedRoundRobin 类型")
}

func TestWeightedRoundRobin_Next(t *testing.T) {
	// 创建测试代理
	proxy1, _ := proxy.New("http://proxy1.test", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy2, _ := proxy.New("http://proxy2.test", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy3, _ := proxy.New("http://proxy3.test", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})

	tests := []struct {
		name    string
		proxies map[*proxy.Proxy]int32
		calls   int
		want    []*proxy.Proxy
	}{
		{
			name: "基本权重测试",
			proxies: map[*proxy.Proxy]int32{
				proxy1: 2,
				proxy2: 1,
			},
			calls: 6,
			want: []*proxy.Proxy{
				proxy1, // weight 2, first call
				proxy1, // weight 2, second call
				proxy2, // weight 1
				proxy1, // weight 2, cycle repeats
				proxy1,
				proxy2,
			},
		},
		{
			name: "多代理权重测试",
			proxies: map[*proxy.Proxy]int32{
				proxy1: 3,
				proxy2: 2,
				proxy3: 1,
			},
			calls: 12,
			want: []*proxy.Proxy{
				proxy1, // weight 3
				proxy1,
				proxy1,
				proxy2, // weight 2
				proxy2,
				proxy3, // weight 1
				proxy1, // 循环重复
				proxy1,
				proxy1,
				proxy2,
				proxy2,
				proxy3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrr := NewWeightedRoundRobin(tt.proxies)

			for i := 0; i < tt.calls; i++ {
				got, err := wrr.Next()
				assert.NoError(t, err)
				assert.Equal(t, tt.want[i], got, "第 %d 次调用返回的代理不符合预期", i+1)
			}
		})
	}
}

func TestWeightedRoundRobin_Next_WithUnavailableProxy(t *testing.T) {
	// 创建测试代理并设置状态
	proxy1, _ := proxy.New("http://proxy1.test", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy2, _ := proxy.New("http://proxy2.test", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	proxy3, _ := proxy.New("http://proxy3.test", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})

	proxies := map[*proxy.Proxy]int32{
		proxy1: 1,
		proxy2: 1,
		proxy3: 1,
	}

	wrr := NewWeightedRoundRobin(proxies)

	// 测试是否正确返回代理
	for i := 0; i < 4; i++ {
		got, err := wrr.Next()
		assert.NoError(t, err)
		assert.NotNil(t, got, "返回的代理不应为 nil")
	}
}
