package proxy

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// 创建一个测试服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tests := []struct {
		name         string
		upstream     string
		healthCheck  HealthCheck
		clientConfig ClientConfig
		wantErr      bool
	}{
		{
			name:     "valid proxy creation",
			upstream: ts.URL,
			healthCheck: HealthCheck{
				Enabled:  true,
				Method:   "GET",
				Path:     "/health",
				Interval: 1,
				Timeout:  1,
			},
			clientConfig: ClientConfig{
				DialTimeout:     time.Second,
				MaxConnsPerHost: 100,
				KeepAlive:       true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.upstream, tt.healthCheck, tt.clientConfig)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, p)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, p)
			}
		})
	}
}

func TestProxy_IsAvailable(t *testing.T) {
	t.Skip()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p, err := New(ts.URL, HealthCheck{
		Enabled:  true,
		Method:   "GET",
		Path:     "/health",
		Interval: 1,
		Timeout:  1,
	}, ClientConfig{})

	assert.NoError(t, err)

	// 等待健康检查执行
	time.Sleep(2 * time.Second)

	assert.True(t, p.IsAvailable())
}

func TestProxy_GetLoad(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p, err := New(ts.URL, HealthCheck{
		Enabled:  true,
		Method:   "GET",
		Path:     "/health",
		Interval: 1,
		Timeout:  1,
	}, ClientConfig{})

	assert.NoError(t, err)

	// 初始负载应该为0
	assert.Equal(t, 0, p.GetLoad())
}

func TestProxy_HealthChecking(t *testing.T) {
	t.Skip()
	// 创建一个测试服务器，可以控制返回状态码
	var statusCode int32 = http.StatusOK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(atomic.LoadInt32(&statusCode)))
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	p, err := New(ts.URL, HealthCheck{
		Enabled:          true,
		Method:           "GET",
		Path:             "/health",
		Interval:         1,
		Timeout:          1,
		AllowStatusCodes: []string{"2xx"},
	}, ClientConfig{
		DialTimeout:     time.Second,
		MaxConnsPerHost: 100,
		KeepAlive:       true,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
	})

	assert.NoError(t, err)
	assert.NotNil(t, p)

	// 等待健康检查完成
	time.Sleep(2 * time.Second)

	// 测试正常状态
	assert.True(t, p.healthChecking())

	// 测试异常状态
	atomic.StoreInt32(&statusCode, http.StatusInternalServerError)
	time.Sleep(time.Second)
	assert.False(t, p.healthChecking())

	// 测试特定状态码
	p.healthCheck.AllowStatusCodes = []string{"500"}
	time.Sleep(time.Second)
	assert.True(t, p.healthChecking())
}
