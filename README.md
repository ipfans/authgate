# AuthGate

AuthGate 是一个用于管理用户认证和授权的系统。它提供了一个简单的 API 接口，用于验证用户的身份，并生成 JWT 令牌。

## 配置

配置文件 `config.yaml` 用于配置 AuthGate 的行为。

```yaml
addr: ":8080"
routes:
  # 路由配置项
  auth_host: "auth.example.com"
  ssl: true
  jwt_secret: "your-jwt-secret-key-here"

  # Cookie 配置
  cookies:
    name: "authgate_token"
    max_age: 86400 # 24小时
    secure: false
    http_only: true
    path: "/"
    domain: ""

  # 认证凭据
  credential:
    username: "admin"
    password: "password"

  # 后端服务配置
  backends:
    - host: "backend.example.com"
      load_balance: "round_robin" # 可选: random, round_robin, least_connections, weighted_round_robin
      weight:
        - 1
      upstream:
        - "http://127.0.0.1:8080"
      health_check:
        enabled: true
        interval: "10"
        timeout: "5"
        healthy_threshold: 2
        unhealthy_threshold: 3
      client:
        timeout: "30s"
        keep_alive_timeout: "30s"
        max_idle_conns: 100
        idle_conn_timeout: "90s"
```
