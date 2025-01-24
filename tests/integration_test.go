package tests

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/ipfans/authgate/config"
	"github.com/ipfans/authgate/routers"
	"github.com/ipfans/components/v2/configuration"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试配置文件
const testConfig = `
addr: ":8080"
routes:
  auth_host: "auth.example.com"
  ssl: false
  jwt_secret: "test_secret"
  cookies:
    name: "authgate_token"
    max_age: 60
    secure: false
    http_only: true
    path: "/"
    domain: ""
  credential:
    username: "testuser"
    password: "testpass"
  backends:
    - host: "test.example.com"
      load_balance: "round_robin"
      upstream:
        - "http://127.0.0.1:8081"
`

func loadTestConfig() config.Config {
	var cfg config.Config
	configuration.Load(&cfg, configuration.WithProvider(rawbytes.Provider([]byte(testConfig)), yaml.Parser()))
	return cfg
}

type loginResponse struct {
	Token string `json:"token"`
}

func setupTestServer(t *testing.T) *route.Engine {
	// 写入临时配置文件
	cfg := loadTestConfig()

	// 启动测试服务器
	h := server.Default()
	routers.RegisterRoutes(h, cfg.Routes)
	return h.Engine
}

func TestIntegration(t *testing.T) {
	ts := setupTestServer(t)

	// 测试未授权访问
	t.Run("Unauthorized Access", func(t *testing.T) {
		rec := ut.PerformRequest(ts, "GET", "/api/protected", nil, ut.Header{
			Key:   "Host",
			Value: "test.example.com",
		})
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Equal(t, "http://auth.example.com/authgate/login?host=http%3A%2F%2Ftest.example.com", rec.Header().Get("Location"))
		rec = ut.PerformRequest(ts, "GET", "/api/protected", nil, ut.Header{
			Key:   "Host",
			Value: "test.example.com",
		}, ut.Header{
			Key:   "X-Forwarded-Proto",
			Value: "https",
		})
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Equal(t, "http://auth.example.com/authgate/login?host=https%3A%2F%2Ftest.example.com", rec.Header().Get("Location"))
	})

	t.Run("Login failed", func(t *testing.T) {
		formData := map[string][]string{
			"username": {"admin"},
			"password": {"admin"},
			"host":     {"http://test.example.com"},
		}
		query := url.Values(formData)
		length := len(query.Encode())
		body := &ut.Body{
			Body: strings.NewReader(query.Encode()),
			Len:  length,
		}
		rec := ut.PerformRequest(ts, "POST", "/login", body, ut.Header{
			Key:   "Host",
			Value: "auth.example.com",
		})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	// 测试登录
	t.Run("Login", func(t *testing.T) {
		var rec *ut.ResponseRecorder
		// 测试登录，账号密码正确
		formData := map[string][]string{
			"username": {"testuser"},
			"password": {"testpass"},
			"host":     {"http://test.example.com"},
		}
		query := url.Values(formData)
		body := &ut.Body{
			Body: strings.NewReader(query.Encode()),
			Len:  len(query.Encode()),
		}
		rec = ut.PerformRequest(ts, "POST", "/login", body, ut.Header{
			Key:   "Host",
			Value: "auth.example.com",
		}, ut.Header{
			Key:   "X-Forwarded-Proto",
			Value: "https",
		}, ut.Header{
			Key:   "Content-Type",
			Value: "application/x-www-form-urlencoded",
		})
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "http://test.example.com/authgate/login/finish?token=eyJh")

		// 测试完成登录
		loc, _ := url.Parse(rec.Header().Get("Location"))
		token := loc.Query().Get("token")
		rec = ut.PerformRequest(ts, "GET", "/authgate/login/finish?token="+url.QueryEscape(token), nil, ut.Header{
			Key:   "Host",
			Value: "test.example.com",
		})
		assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		assert.Equal(t, "/", rec.Header().Get("Location"))
		cookies := rec.Result().Header.GetCookies()
		// fmt.Println(cookies)
		jwtToken := ""
		containsCookie := false
		for _, cookie := range cookies {
			if string(cookie.GetKey()) == "authgate_token" {
				containsCookie = true
				// 解析字符串：authgate_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mzc3Nzc1MTYsInVzZXJuYW1lIjoidGVzdHVzZXIifQ.FO86HUlHsCaYGa7ER1Mb1GAlFmzvyFJJVTpPZMpFvBw; max-age=60; path=/; HttpOnly; SameSite
				rawString := string(cookie.GetValue())
				parts := strings.Split(rawString, ";")
				for _, part := range parts {
					if strings.HasPrefix(part, "authgate_token=") {
						jwtToken = strings.TrimPrefix(part, "authgate_token=")
						break
					}
				}
			}
		}
		assert.True(t, containsCookie, "authgate_token cookie not found")
		assert.NotEmpty(t, jwtToken, "jwt token not found")

		// 测试使用token访问受保护资源
		rec = ut.PerformRequest(ts, "GET", "/api/protected", nil, ut.Header{
			Key:   "Host",
			Value: "test.example.com",
		}, ut.Header{
			Key:   "Cookie",
			Value: fmt.Sprintf("authgate_token=%s", jwtToken),
		})
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
