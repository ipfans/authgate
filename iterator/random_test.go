package iterator

import (
	"testing"

	"github.com/ipfans/authgate/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRandom(t *testing.T) {
	// 准备测试数据
	proxies := make([]*proxy.Proxy, 0, 3)
	for _, addr := range []string{"http://proxy1:8080", "http://proxy2:8080", "http://proxy3:8080"} {
		p, err := proxy.New(addr, proxy.HealthCheck{
			Enabled: false,
		}, proxy.ClientConfig{})
		require.NoError(t, err)
		proxies = append(proxies, p)
	}

	// 测试创建实例
	seed := func() {}
	iterator := NewRandom(seed, proxies...)

	// 验证返回的迭代器类型
	_, ok := iterator.(*Random)
	assert.True(t, ok, "应该返回 Random 类型的迭代器")
}

func TestRandom_Next(t *testing.T) {
	tests := []struct {
		name    string
		proxies []*proxy.Proxy
		wantErr bool
		errMsg  string
	}{
		{
			name: "有效代理列表",
			proxies: func() []*proxy.Proxy {
				proxies := make([]*proxy.Proxy, 0, 3)
				for _, addr := range []string{"http://proxy1:8080", "http://proxy2:8080", "http://proxy3:8080"} {
					p, err := proxy.New(addr, proxy.HealthCheck{
						Enabled: false,
					}, proxy.ClientConfig{})
					require.NoError(t, err)
					proxies = append(proxies, p)
				}
				return proxies
			}(),
			wantErr: false,
		},
		{
			name:    "空代理列表",
			proxies: []*proxy.Proxy{},
			wantErr: true,
			errMsg:  "no proxies set",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// 创建迭代器实例
			r := &Random{
				proxies: make(commonProxiesBunch, 0, len(tt.proxies)),
			}
			for _, p := range tt.proxies {
				r.proxies = append(r.proxies, p)
			}

			// 测试 Next 方法
			proxy, err := r.Next()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				assert.Nil(t, proxy)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, proxy)
				// 验证返回的代理是否在原始列表中
				found := false
				for _, p := range tt.proxies {
					if p == proxy {
						found = true
						break
					}
				}
				assert.True(t, found, "返回的代理应该在原始代理列表中")
			}
		})
	}
}

func TestRandom_Distribution(t *testing.T) {
	// 准备测试数据
	proxies := make([]*proxy.Proxy, 0, 3)
	for _, addr := range []string{"http://proxy1:8080", "http://proxy2:8080", "http://proxy3:8080"} {
		p, err := proxy.New(addr, proxy.HealthCheck{
			Enabled: false,
		}, proxy.ClientConfig{})
		require.NoError(t, err)
		proxies = append(proxies, p)
	}

	// 创建迭代器
	r := &Random{
		proxies: make(commonProxiesBunch, 0, len(proxies)),
	}
	for _, p := range proxies {
		p := p // capture range variable
		r.proxies = append(r.proxies, p)
	}

	// 统计分布
	iterations := 1000
	distribution := make(map[*proxy.Proxy]int)

	for i := 0; i < iterations; i++ {
		proxy, err := r.Next()
		assert.NoError(t, err)
		distribution[proxy]++
	}

	// 验证每个代理都被使用过
	for _, p := range proxies {
		p := p // capture range variable
		count := distribution[p]
		assert.Greater(t, count, 0, "每个代理都应该被使用过")
	}
}
