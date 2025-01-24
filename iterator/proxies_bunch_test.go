package iterator

import (
	"testing"

	"github.com/ipfans/authgate/proxy"
	"github.com/stretchr/testify/assert"
)

func createTestProxy(available bool) *proxy.Proxy {
	p, _ := proxy.New("http://test.proxy", proxy.HealthCheck{
		Enabled: false,
	}, proxy.ClientConfig{})
	p.SetAvailable(available)
	return p
}

func TestGetAvailableProxy(t *testing.T) {
	tests := []struct {
		name    string
		proxies proxiesBunch
		marker  int
		want    int // 期望返回的代理索引
		wantErr bool
	}{
		{
			name: "找到第一个可用代理",
			proxies: commonProxiesBunch{
				createTestProxy(true),
				createTestProxy(false),
				createTestProxy(false),
			},
			marker: 0,
			want:   0,
		},
		{
			name: "从marker开始找到可用代理",
			proxies: commonProxiesBunch{
				createTestProxy(false),
				createTestProxy(true),
				createTestProxy(false),
			},
			marker: 1,
			want:   1,
		},
		{
			name: "循环查找可用代理",
			proxies: commonProxiesBunch{
				createTestProxy(true),
				createTestProxy(false),
				createTestProxy(false),
			},
			marker: 2,
			want:   0,
		},
		{
			name: "所有代理不可用",
			proxies: commonProxiesBunch{
				createTestProxy(false),
				createTestProxy(false),
			},
			marker:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAvailableProxy(tt.proxies, tt.marker)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.proxies.Get(tt.want), got)
		})
	}
}

func TestCommonProxiesBunch(t *testing.T) {
	proxies := commonProxiesBunch{
		createTestProxy(true),
		createTestProxy(false),
	}

	assert.Equal(t, 2, proxies.Len())
	assert.True(t, proxies.Get(0).IsAvailable())
	assert.False(t, proxies.Get(1).IsAvailable())
}

func TestWeightedProxiesBunch(t *testing.T) {
	proxies := weightedProxiesBunch{
		{Proxy: createTestProxy(true), weight: 2},
		{Proxy: createTestProxy(false), weight: 1},
	}

	assert.Equal(t, 2, proxies.Len())
	assert.True(t, proxies.Get(0).IsAvailable())
	assert.False(t, proxies.Get(1).IsAvailable())
}
