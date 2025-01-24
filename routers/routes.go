package routers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/golang-jwt/jwt/v5"
	"github.com/ipfans/authgate/iterator"
	"github.com/ipfans/authgate/proxy"
	"github.com/rs/zerolog/log"
)

type Backend struct {
	Host         string             `koanf:"host"`
	LoadBalance  string             `koanf:"load_balance"`
	Weight       []int32            `koanf:"weight"`
	UpStream     []string           `koanf:"upstream"`
	HealthCheck  proxy.HealthCheck  `koanf:"health_check"`
	ClientConfig proxy.ClientConfig `koanf:"client"`
}

type CookieConfig struct {
	Name     string `koanf:"name"`
	MaxAge   int    `koanf:"max_age"`
	Secure   bool   `koanf:"secure"`
	HttpOnly bool   `koanf:"http_only"`
	Path     string `koanf:"path"`
	Domain   string `koanf:"domain"`
}

type CredentialConfig struct {
	Username string `koanf:"username"`
	Password string `koanf:"password"`
}

type Config struct {
	Backends   []Backend        `koanf:"backends"`
	AuthHost   string           `koanf:"auth_host"`
	SSL        bool             `koanf:"ssl"`
	JWTSecret  string           `koanf:"jwt_secret"`
	Cookies    CookieConfig     `koanf:"cookies"`
	Credential CredentialConfig `koanf:"credential"`
}

func RegisterRoutes(e *server.Hertz, cfg Config) (err error) {
	backends := make(map[string]iterator.Iterator, len(cfg.Backends))
	for _, backend := range cfg.Backends {
		proxies := make([]*proxy.Proxy, 0, len(backend.UpStream))
		for _, upstream := range backend.UpStream {
			p, err := proxy.New(upstream, backend.HealthCheck, backend.ClientConfig)
			if err != nil {
				return err
			}
			proxies = append(proxies, p)
		}
		// 根据配置选择负载均衡策略
		var it iterator.Iterator
		switch backend.LoadBalance {
		case "random":
			it = iterator.NewRandom(nil, proxies...)
		case "round_robin":
			it = iterator.NewRoundRobin(proxies...)
		case "least_connections":
			it = iterator.NewLeastConnections(proxies...)
		case "weighted_round_robin":
			// 创建权重映射
			proxyWeights := make(map[*proxy.Proxy]int32)
			for i, p := range proxies {
				weight := int32(1) // 默认权重为1
				if i < len(backend.Weight) {
					weight = backend.Weight[i]
				}
				proxyWeights[p] = weight
			}
			it = iterator.NewWeightedRoundRobin(proxyWeights)
		default:
			// 默认使用随机策略
			it = iterator.NewRoundRobin(proxies...)
		}
		backends[backend.Host] = it
	}

	proxyFunc := func(ctx context.Context, c *app.RequestContext) {
		rp, ok := backends[string(c.Host())]
		if !ok {
			c.Header("X-Error", "No backend found")
			c.Status(http.StatusNotFound)
			return
		}
		proxy, err := rp.Next()
		if err != nil {
			log.Error().Err(err).Msg("No backend found")
			c.Header("X-Error", "No backend found")
			c.String(http.StatusServiceUnavailable, "Internal Server Error")
			return
		}
		proxy.ServeHTTP(ctx, c)
	}

	allowMiddleware := func(ctx context.Context, c *app.RequestContext) {
		host := string(c.Host())
		if host != cfg.AuthHost {
			proxyFunc(ctx, c)
			return
		}
		c.Next(ctx)
	}

	authCheckMiddleware := func(ctx context.Context, c *app.RequestContext) {
		token := string(c.Cookie(cfg.Cookies.Name))
		prefix := "http://"
		if cfg.SSL {
			prefix = "https://"
		}
		query := url.Values{}
		targetHost := string(c.Host())
		proto := string(c.GetHeader("X-Forwarded-Proto"))
		if proto == "" {
			proto = "http"
		}
		targetHost = proto + "://" + targetHost
		query.Add("host", targetHost)
		host := prefix + cfg.AuthHost + "/authgate/login?" + query.Encode()
		if token == "" {
			c.Redirect(http.StatusTemporaryRedirect, []byte(host))
			return
		}

		_, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, []byte(host))
			return
		}
		c.Next(ctx)
	}

	e.NoRoute(authCheckMiddleware, func(ctx context.Context, c *app.RequestContext) {
		host := string(c.GetRequest().Header.Get("Host"))
		if host == cfg.AuthHost {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		token := string(c.Cookie(cfg.Cookies.Name))
		prefix := "http://"
		if cfg.SSL {
			prefix = "https://"
		}
		query := url.Values{}
		targetHost := host
		proto := string(c.GetHeader("X-Forwarded-Proto"))
		if proto == "" {
			proto = "http"
		}
		targetHost = proto + "://" + targetHost
		query.Add("host", targetHost)
		host = prefix + cfg.AuthHost + "/authgate/login?" + query.Encode()
		if token == "" {
			c.Redirect(http.StatusTemporaryRedirect, []byte(host))
			return
		}

		_, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, []byte(host))
			return
		}

		proxyFunc(ctx, c)
	})

	e.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(http.StatusOK, "AuthGate is running...")
	}, allowMiddleware)

	e.GET("/login", allowMiddleware, func(ctx context.Context, c *app.RequestContext) {
		host := c.Query("host")
		c.HTML(http.StatusOK, "login.html", utils.H{
			"host": host,
		})
	})

	e.POST("/login", allowMiddleware, func(ctx context.Context, c *app.RequestContext) {
		var host, token string

		host = c.PostForm("host")
		username := c.PostForm("username")
		password := c.PostForm("password")
		if username != cfg.Credential.Username || password != cfg.Credential.Password {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		tokenData := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
		var err error
		token, err = tokenData.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		query := url.Values{}
		query.Add("token", token)
		c.Redirect(http.StatusTemporaryRedirect, []byte(host+"/authgate/login/finish?"+query.Encode()))
	})

	e.GET("/authgate/login/finish", func(ctx context.Context, c *app.RequestContext) {
		host := string(c.GetRequest().Header.Get("Host"))
		if host == cfg.AuthHost {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		token := c.Query("token")
		if token == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		_, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.SetCookie(
			cfg.Cookies.Name,
			token,
			cfg.Cookies.MaxAge,
			cfg.Cookies.Path,
			cfg.Cookies.Domain,
			protocol.CookieSameSiteDefaultMode,
			cfg.Cookies.Secure,
			cfg.Cookies.HttpOnly,
		)
		c.Redirect(http.StatusTemporaryRedirect, []byte("/"))
	})

	return
}
