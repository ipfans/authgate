package iterator

import (
	"testing"

	"github.com/ipfans/authgate/proxy"
	"github.com/stretchr/testify/assert"
)

func TestNewLeastConnections(t *testing.T) {
	// 创建测试代理
	p1 := &proxy.Proxy{}
	p2 := &proxy.Proxy{}

	// 测试创建新的最小连接数迭代器
	lc := NewLeastConnections(p1, p2)

	assert.NotNil(t, lc, "应该成功创建LeastConnections实例")
	assert.IsType(t, &LeastConnections{}, lc, "应该返回LeastConnections类型")
}

func TestLeastConnections_Next(t *testing.T) {
	tests := []struct {
		name        string
		proxies     []*proxy.Proxy
		loads       []int32
		wantErr     bool
		expectedIdx int
	}{
		{
			name:        "空代理列表",
			proxies:     []*proxy.Proxy{},
			loads:       []int32{},
			wantErr:     true,
			expectedIdx: -1,
		},
		{
			name:        "单个代理",
			proxies:     []*proxy.Proxy{{}},
			loads:       []int32{1},
			wantErr:     false,
			expectedIdx: 0,
		},
		{
			name:        "多个代理-选择负载最小的",
			proxies:     []*proxy.Proxy{{}, {}, {}},
			loads:       []int32{3, 1, 2},
			wantErr:     false,
			expectedIdx: 1, // 负载为1的代理应该被选中
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置每个代理的负载
			for i, p := range tt.proxies {
				// 设置负载值和可用性
				p.SetLoad(int(tt.loads[i]))
				p.SetAvailable(true)
			}

			// 创建迭代器
			lc := NewLeastConnections(tt.proxies...)

			// 获取下一个代理
			got, err := lc.Next()

			if tt.wantErr {
				assert.Error(t, err, "应该返回错误")
				assert.Nil(t, got, "错误情况下应该返回nil")
			} else {
				assert.NoError(t, err, "不应该返回错误")
				assert.NotNil(t, got, "应该返回一个代理")

				if tt.expectedIdx >= 0 {
					assert.Equal(t, tt.proxies[tt.expectedIdx], got,
						"应该返回负载最小的代理")
				}
			}
		})
	}
}

func TestLeastConnections_Next_WithUnavailableProxies(t *testing.T) {
	// 创建代理
	p1 := &proxy.Proxy{}
	p2 := &proxy.Proxy{}
	p3 := &proxy.Proxy{}
	// 设置负载
	p1.SetLoad(1)
	p2.SetLoad(2)
	p3.SetLoad(1)

	// 设置可用性
	p1.SetAvailable(false)
	p2.SetAvailable(false)
	p3.SetAvailable(true)

	// 创建迭代器
	lc := NewLeastConnections(p1, p2, p3)

	// 获取下一个代理
	got, err := lc.Next()

	assert.NoError(t, err, "不应该返回错误")
	assert.Equal(t, p3, got, "应该返回唯一可用的代理p3")
}
