# 系统运维与维护手册

本手册面向 Aofei / Winter DSP 的系统运维人员、值班人员和代码维护者。它把生产部署、日常任务、缓存、日志账务、监控、故障处理和变更验证汇总成一个中文入口。生产环境的主机清单、域名、证书、凭据、备份平台和告警系统仍由具体部署维护。

## 1. 系统边界

两个相邻代码库共同组成当前服务：

| 代码库 | 责任 |
|---|---|
| `aofei` | DSP/SSP 竞价、匹配、测量、缓存编译、日志与账务作业、MySQL 基线和本地 Docker 环境。 |
| `../pzdesign` | `cmd/unify` HTTP 服务、Summer/Genelet 管理 UI、模板和静态资源。 |

生产依赖：

- MySQL：账户、流量资源、投放、路由配置和账务的事实来源；
- Redis：共享可变状态、Redis 模式静态缓存、外部需求方（内部名 `middleman`）路由与回调上下文；
- NATS：尽力而为的审计日志传输和可选 spread 缓存分发；
- 外部 GeoLite2 City `.mmdb`：IP 地理信息；
- 持久化日志、上传和 spread 目录。

生产目标是 Linux + systemd。仓库中的 Docker Compose 辅助环境仅用于本地开发，不是生产部署契约。

## 2. 进程与节点职责

| 节点/职责 | 进程或任务 | 约束 |
|---|---|---|
| HTTP/UI/ADX 节点 | `unify` | 每个 HTTP 节点运行；处理管理 UI、`/bid`、`/pz`、测量和外部需求方回调。 |
| 日志写入/聚合节点 | `nats-client` | 将 NATS 日志写成时间片文件。建议完整 win/loss 流汇聚到一个账务节点。 |
| 缓存维护节点 | `redis-cache` 定时任务 | 只能单例运行，不能在每台 HTTP 节点分别刷新。 |
| 账务聚合节点 | `ledger` 时间片和每日任务 | 只能在拥有完整 `log_winloss` 流的节点运行。 |
| 外部需求方回调重试节点 | `mid-callback-retry` 定时任务 | 单例运行；只转发下游回调，不重复发布账务事件。 |
| 本地静态缓存节点 | `spread` | 仅在采用 spread/local 静态缓存的节点运行。 |
| 维护节点 | `maxmind`、`winloss` | MaxMind 可定时或人工执行；win/loss 模拟器只用于测试。 |

`redis-cache`、`ledger`、`mid-callback-retry` 和 `winloss` 的变更模式默认取得 Redis 单例锁。所有 `redis-cache` 写模式共享 `aofei:redis-cache` 锁，完整刷新与路由局部刷新不能重叠。

## 3. 生产文件与权限

推荐布局：

```text
/etc/aofei/aofei.json
/etc/aofei/summer.json
/opt/aofei/bin/unify
/opt/aofei/bin/nats-client
/opt/aofei/bin/spread
/opt/aofei/bin/ledger
/opt/aofei/bin/maxmind
/opt/aofei/bin/redis-cache
/opt/aofei/bin/mid-callback-retry
/var/lib/aofei/uploads
/var/lib/aofei/spread
/var/lib/aofei/maxmind/GeoLite2-City.mmdb
/var/log/aofei/log_request
/var/log/aofei/log_response
/var/log/aofei/log_attribute
/var/log/aofei/log_winloss
```

创建专用 `aofei` 用户和组：配置只读、二进制可执行、上传/spread/日志目录可写，代码仓库不可写。`nats-client` 会把日志目录创建或收紧为 `0750`，日志文件为 `0640`；不要把账务输入设为全局可读或可写。

上传目录应位于公开静态根目录之外，除非经过单独安全评审。模板和静态资产应来自与 `unify` 二进制匹配的 `pzdesign` 发布物，模板对服务用户只读。

## 4. 配置与密钥

`unify` 同时读取：

```text
AOFEI=/etc/aofei/aofei.json
SUMMER=/etc/aofei/summer.json
```

其他 Aofei 命令读取 `AOFEI`。DSP 配置使用小写 JSON 字段，Summer/Genelet 配置使用首字母大写字段；不能混用。仓库中的 `etc/aofei.json` 和 `etc/summer.example.json` 只是示例，本地生成的 `etc/*.local.json` 也不能原样复制到生产。

不得提交数据库密码、Redis/NATS 凭据、SMTP 凭据、Session/OAuth 密钥、云密钥、跟踪密钥或真实外部需求方请求头。建议由部署系统或 root 拥有的环境文件注入。

重点配置：

- `tracking_secret` 或环境变量 `TRACKING_SECRET`：签名 win/loss、展示、点击重定向和外部需求方回调；生产必须设置稳定且受保护的值；
- `tracking_signature_ttl_seconds`：签名有效期，默认 86400 秒；接收端还接受最多 5 分钟未来时钟偏差；
- `cap_state_ttl_seconds`：用户频控状态空闲 TTL，默认 90 天；更新不会缩短更长的现有 TTL；
- `middleman_enabled`：外部 DSP / ADX 需求方竞价总开关；
- `middleman_always_enabled`：`Always` 路由独立开关，默认关闭；
- `middleman_exchange_domain`、`middleman_timeout_ms`、`middleman_max_bidders_per_imp`：转发身份与预算；
- `middleman_route_cache_ttl_ms`：HTTP 进程内路由快照/错误缓存时间，默认 5 秒；
- `middleman_callback_base_url`、回调 TTL 和回调超时：代理回调的公网地址和生命周期；
- `trusted_proxy_cidrs`：仅列出真正受控的反向代理；否则客户端可伪造 IP；
- Summer `ServerURL` 与 `CORSOrigins`：管理 UI 仅允许完全匹配的来源；
- Summer 数据库登录签发器：使用 `Password_hash: "passwd"` 校验 bcrypt 密码，并按查询返回列的顺序完整配置 `OutPars`，其中必须包含 `passwd`。

每个活动中的 `adv_bidder.credential_ref` 只能保存环境变量名称。该变量对 `unify` 可见，值为 JSON HTTP 请求头对象，例如 `{"Authorization":"Bearer ..."}`。真实值不得写入 MySQL、Redis、管理页面或版本库。系统会拒绝 `Host`、`Connection`、`Content-Length` 等逐跳或不安全头。

## 5. 本地开发环境

从 `aofei` 仓库运行：

```bash
./scripts/aofei-local.sh up
./scripts/aofei-local.sh reset-sample
./scripts/aofei-local.sh status
```

默认服务：MySQL `127.0.0.1:3307`、Redis `127.0.0.1:6379`、NATS `127.0.0.1:4222`。辅助脚本生成忽略提交的：

```text
etc/aofei.local.json
etc/summer.local.json
```

填充 Redis 并启动统一服务：

```bash
GOWORK=off AOFEI="$PWD/etc/aofei.local.json" \
  go run ./cmd/redis-cache -cache=redis

(cd ../pzdesign && \
  GOWORK=off \
  AOFEI="$PWD/../aofei/etc/aofei.local.json" \
  SUMMER="$PWD/../aofei/etc/summer.local.json" \
  go run ./cmd/unify)
```

停止容器但保留数据卷：

```bash
./scripts/aofei-local.sh down
```

除非明确修改父级 workspace，所有仓库内 Go 命令都使用 `GOWORK=off`。

## 6. 构建、发布与停止

从已评审的提交构建。至少先运行两仓库全量测试，再安装二进制：

```bash
GOWORK=off go test ./...
GOWORK=off go install ./cmd/nats-client ./cmd/spread ./cmd/ledger \
  ./cmd/maxmind ./cmd/redis-cache ./cmd/mid-callback-retry

(cd ../pzdesign && \
  GOWORK=off go test ./... && \
  GOWORK=off go install ./cmd/unify)
```

把二进制复制到带版本的发布目录，再切换活动软链接或 systemd `ExecStart`。先部署辅助进程，再逐台滚动重启 `unify`。常见操作：

```bash
sudo systemctl daemon-reload
sudo systemctl restart aofei-spread.service
sudo systemctl restart aofei-nats-client.service
sudo systemctl restart aofei-unify.service
sudo systemctl enable --now aofei-redis-cache.timer
sudo systemctl enable --now aofei-mid-callback-retry.timer
sudo systemctl enable --now aofei-ledger.timer
sudo systemctl enable --now aofei-ledger-daily.timer
sudo systemctl status aofei-unify.service
```

`unify` 收到 `SIGINT` 或 `SIGTERM` 后停止接受新请求，最多等待 15 秒让在途 HTTP 请求结束，然后排空审计发布队列并依次关闭 NATS、Redis 和 MySQL。systemd 的停止超时必须大于 15 秒，不能在应用排空前发送强制终止。

发布后检查：

- 管理页面和静态资源可访问；
- `/bid/<测试域名>` 与 `/pz` 测试请求返回预期结果；
- `/debug/vars` 可抓取且关键计数器没有异常跳升；
- Redis 静态缓存和路由生成时间正确；
- NATS 日志开始写入；
- 外部需求方回调重试与账务定时器仅在指定单例节点运行。

发生启动失败、持续竞价错误、缓存反序列化错误或账务写入错误时，恢复上一版本二进制并重启同一组服务。数据库变更必须使用预先准备的逆向方案或备份恢复，不能只回滚二进制。

## 7. 静态缓存操作

MySQL 是事实来源。`redis-cache` 把 MySQL 数据编译为运行缓存。完整 Redis 刷新使用影子键构建一个完整世代，成功后在一个 Redis 事务中切换；构建失败会保留旧世代，读请求不会看到先删除后回填的窗口。

生产缓存节点执行：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/redis-cache -cache=redis
```

常见模式：

| 模式 | 用途 |
|---|---|
| `-cache=redis` | 重建 Redis 全部静态缓存和外部需求方路由。 |
| `-cache=spread` | 通过 NATS 发布 spread/local 静态快照。 |
| `-cache=all` | 同时执行 spread 和 Redis 发布。 |
| `-cache=routes` | 只刷新外部需求方路由，不改其他缓存族。 |
| `-cache=routes -read` | 只读路由 JSON 与元数据，不取得写锁。 |

关键 Redis 静态缓存族：

```text
pubmap
pubmap:by-id
audience
creative
slot:<size_id>
middleman:routes:v2
middleman:routes
```

`middleman:routes:v2` 是当前优先格式；旧 `middleman:routes` 仅用于滚动发布兼容。管理 UI 中路由组、路由竞价方或目标的编辑只写 MySQL，不会自动刷新 Redis。编辑后必须在单例缓存节点运行完整或 route-only 刷新，并等待各 HTTP 进程的短期路由快照过期。

路由刷新失败时，HTTP 进程会在短时间内缓存错误并禁用外部需求方扇出，不会继续使用过期路由授权流量；本地已有赢家仍按正常规则保留。即使取消了发起刷新请求的客户端，共享加载也使用独立超时继续服务其他等待者。

以下 Redis 数据是共享可变状态，静态缓存维护不能误删：

```text
bothcap:<user_id>
upload:<adv_id>:<marker>
middleman:cb:<token>
middleman:bill:<token>
跟踪事件重复抑制和频控事件标记
```

`bothcap` 更新保证至少保留配置 TTL，并且不会缩短更长 TTL。展示/点击重复标记的过期时间对应签名的确切有效截止点，最长可能是配置 TTL 再加接收端允许的 5 分钟未来时钟偏差。

### spread/local 模式

`spread` 将 NATS 缓存消息原子写入：

```text
<spread>/pubmap/
<spread>/audience/
<spread>/creative/
<spread>/slot/<size_id>/
```

DSP 在启动时把这些文件载入内存，热请求不读取文件系统。后续必须通过显式重载钩子加载新快照，或按发布流程重启节点。外部需求方路由仍是 Redis-only；即使本地静态投放不依赖 Redis，启用外部需求方竞价的节点仍需 Redis。

本地缓存过旧不会自动停止竞价。必须监控缓存年龄和 stale 指标，并在告警后重载或重启受影响节点。

## 8. 日志、账务与外部需求方回调重试

### 8.1 NATS 日志写入

`unify` 发布 `request`、`response`、`attribute` 和 `winloss`。`nats-client` 写入：

```text
log_request/request.<stamp>
log_response/response.<stamp>
log_attribute/attribute.<stamp>
log_winloss/winloss.<stamp>
```

请求/响应/属性审计在 HTTP 响应后通过有界后台队列尽力发布。NATS 故障或队列溢出不会撤回已返回的竞价结果，所以审计缺失必须通过指标告警，不能假设响应成功就一定有审计日志。

### 8.2 账务

时间片聚合：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/ledger -interval=10
```

指定时间片重放：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/ledger -interval=10 -timestamp=<stamp>
```

每日聚合：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/ledger -daily -timestamp=2026-05-12
```

必须等相应日志轮转窗口关闭后再跑时间片任务，并在当天所有时间片完成后跑每日任务。缺少 `winloss.<stamp>` 是可重试输入错误，不是“零数据成功”。常规账务以 `StatusTrackImp` 作为展示和花费、`StatusTrackClk` 作为点击；裸 win/loss 只是分析事件。

### 8.3 外部需求方回调重试

只读检查和稳定 JSON 摘要：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/mid-callback-retry -read -json
```

处理：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/mid-callback-retry -limit=100 -max-attempts=5 -timeout=2s
```

稳定字段为 `due`、`stale_processing`、`selected`、`succeeded`、`retrying` 和 `abandoned`。持续非零的 `stale_processing` 或增长快于处理速度的 `due` 需要告警。

只对网络错误、HTTP 429 和 5xx 等可重试的 win/loss/bill 下游转发失败入队。非法/缺失 URL、重复通知和除 429 外的 4xx 不入队。重试任务会拒绝环回、私网、链路本地、未指定、多播和 DNS 重绑定目标。

## 9. 数据库与 MaxMind

### 9.1 数据库生命周期

当前本地模式的活动基线为 `etc/step4_init.sql`。新环境先装入经评审的基线，再执行部署方管理的生产迁移。每次 schema 变更前：

- 备份 MySQL；
- 记录当前二进制版本和迁移版本；
- 在具有代表数据的预发布库执行；
- 验证管理页面、缓存编译和竞价；
- 为回滚准备逆向迁移或已验证恢复点。

本地维护基线的顺序：

```bash
./scripts/aofei-local.sh reset
./scripts/aofei-local.sh load
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh diff-schema
```

有意修改 schema 后，必须同步 `etc/step4_init.sql`、`etc/step5.notes`、数据库文档和相关 memory bank。基线不得包含显式旧 `DEFINER` 或旧 MySQL 账户。

生产恢复不以“数据已导入”为结束。恢复 MySQL 后还必须重新编译 Redis/本地静态缓存，并完成 HTTP、管理 UI、竞价和账务 smoke。

### 9.2 MaxMind

真实 `.mmdb` 是外部资产，不得提交。更新运行 JSON：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/maxmind -city=/var/lib/aofei/maxmind/GeoLite2-City.mmdb
```

命令从 MySQL 读取国家/地区 ID 映射，并原子替换配置的 `ips` JSON；它只记录 `.mmdb` 路径，不复制文件，也不验证数据文件内容。上线前应额外检查文件存在、服务用户可读，并用实际 IP 做地理查找 smoke。

## 10. 安全检查

- 生产密钥不进入代码库、MySQL、Redis、页面或工单明文；
- 管理 UI CORS 只允许 `ServerURL` 和明确列出的精确来源；
- `/pz` 的预检虽然是无凭据宽松 CORS，但实际 `POST` 会在竞价前验证打包令牌和精确站点来源；
- 只信任 `trusted_proxy_cidrs` 内代理提供的转发 IP；
- 静态路径会清理并限制在 `DocumentRoot` 内，上传目录保持非公开；
- 外部需求方回调出口保留 SSRF/DNS 重绑定防护，不以网络便利为由关闭；
- 老旧 SHA1 账户密码上线前全部重置；
- 配置、模板、GeoIP 和二进制使用版本化发布物并校验来源。

## 11. 监控与告警

`unify` 在 `/debug/vars` 暴露 expvar。至少监控：

| 信号 | 含义与动作 |
|---|---|
| bid/no-bid 与 SSP 结果 | 按入口、发布版本和流量结构建立基线；突变时检查缓存、MySQL 变更和上游请求。 |
| `aofei_audit_dropped_total`、审计发布错误、队列深度 | 持续增长说明 NATS 或消费端故障，已返回的竞价不会回滚。 |
| `aofei_tracking_replay_redis_errors_total` | 跟踪重复抑制 Redis 失败；合法事件仍 fail-open。 |
| `aofei_tracking_replay_unkeyed_total` | 事件缺少完整重复键；会发布但跳过非幂等频控写入。 |
| `aofei_tracking_cap_update_fail_open_total` | 合法展示/点击已发布，但频控更新失败。 |
| `aofei_tracking_replay_fail_open_total` | 重复控制不可用时接受的事件；注意至少一次计量风险。 |
| `aofei_bothcap_refresh_conflicts_total` | 频控并发冲突；持续上升时检查 Redis 延迟和热点用户。 |
| 外部需求方 route cache hit/miss/refresh/error | refresh error 会暂时禁用扇出，且不会复用过期路由。 |
| 外部需求方回调转发结果与 retry `due` | 检查下游端点、网络、429/5xx 和任务吞吐。 |
| `aofei_local_cache_loaded_at_unix`、`aofei_local_cache_age_seconds`、`aofei_local_cache_stale` | 本地静态缓存年龄；stale 时重载或重启。 |
| `aofei_ssp_policy_rejections_total` | `/pz` 来源策略拒绝；结合 403 访问日志检查站点主机和代理。 |

基础设施还应监控 MySQL 连接/慢查询/磁盘、Redis 内存/淘汰/错误/客户端、NATS 可用性/订阅/丢消息、日志磁盘容量、systemd 重启次数、HTTP 延迟和错误率、证书有效期及节点时钟偏差。

## 12. 故障处理

### 12.1 大量无竞价或 no-fill

1. 按 `/bid` 和 `/pz` 区分入口，并确认上游请求形状、域名和尺寸。
2. 检查 `unify` 健康、错误日志和 `/debug/vars`。
3. 检查 `pubmap`/`pubmap:by-id`、`slot:*`、`audience`、`creative` 是否属于当前世代。
4. 检查活动时间、预算、状态、币种、尺寸、ACL、行业和人群定向。
5. 外部需求方流量检查总开关、Always 开关、路由健康、凭据引用、路由缓存生成时间和刷新错误。
6. local/spread 模式检查缓存年龄及节点是否完成重载。

### 12.2 `/pz` 大量 `400` 或 `403`

- `400`：检查 JSON、体积、`site`/`slot` 令牌、尺寸、重复 `code`、媒体类型和缓存是否更新；
- `403`：检查实际 `Origin`/`Referer` 是否存在且主机与缓存站点完全一致，包括子域差异；
- SDK 可以不带来源，但带了错误来源仍会失败；
- 策略拒绝发生在 Cookie、竞价和审计之前，因此不能只查 NATS 审计。

### 12.3 Redis 故障

- 静态 Redis 模式可能失去 publisher、candidate、audience 和 creative 读取；
- local/spread 模式中不使用频控或上传人群的本地投放可继续，但需要这些共享可变状态的候选会 fail-closed；
- 合法 `/imp`、`/clk` 的重复 claim 或 cap Redis 错误会 fail-open 并继续发布，计量可能至少一次；
- 外部需求方竞价依赖 Redis 路由和回调上下文，即使本地静态缓存可用也会受影响；
- Redis 恢复或 failover 后重新发布静态缓存，且不要清除仍需保留的频控/上传/回调键族。

### 12.4 NATS 或日志故障

竞价响应可能继续成功，但请求、响应、属性或 win/loss 日志会缺失。检查审计 drop/publish 指标、NATS 订阅和磁盘。服务恢复后如订阅未自行恢复，重启 `nats-client` 和 `spread`。无法从未持久化的 Core NATS 消息重建丢失审计；应记录影响时间窗和账务风险。

### 12.5 账务缺失或重复风险

- 确认完整 `winloss.<stamp>` 已汇聚到唯一账务节点；
- 不要把缺失文件当零流量；恢复输入后用明确 `-timestamp` 重跑；
- 确认时间片已结束且每日任务在全部时间片之后；
- 外部需求方报表同时检查 `ledger_mid`/`daily_mid` 和回调重试状态；
- 执行任何重放前确认任务自身的事务与单例锁，并保留原始日志备份。

### 12.6 外部需求方路由修改未生效

1. 在管理 UI `midroute?action=health` 检查无 target、无 bidder、竞价方未启用、凭据未批准或合成链非法等问题。
2. 使用 `redis-cache -cache=routes -read` 比较 Redis 元数据与 MySQL 高水位时间。
3. 在单例缓存节点执行 `-cache=routes`。
4. 等待 `middleman_route_cache_ttl_ms`，观察 refresh/error 指标。
5. 不要通过重启所有 HTTP 节点替代路由发布流程。

## 13. 维护变更与验证

代码维护前依次阅读：

1. `memory-bank/product.md`
2. `memory-bank/architecture.md`
3. `memory-bank/tech-stack.md`
4. `memory-bank/milestone.md`
5. 当前里程碑对应的 `memory-bank/status-M*.md`

任何改变当前运行配置、schema、缓存契约或运维流程的变更，都必须在同一变更中更新 memory bank 和相关文档。不要重新创建根级产品/架构/状态文档，也不要把 `backup/` 历史文件当运行输入。

当前完整验证入口：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off staticcheck ./...
GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback \
  ./internal/jobs/cache ./internal/jobs/ledger ./cmd/spread ./cmd/nats-client
GOWORK=off go test ./dsp ./match -run '^$' -bench . -benchmem
./scripts/aofei-doc-check.sh
./scripts/aofei-cache-smoke.sh
./scripts/aofei-local.sh check-sql
./scripts/aofei-local.sh diff-schema
git diff --check

(cd ../pzdesign && GOWORK=off go test ./...)
(cd ../pzdesign && GOWORK=off go vet ./...)
(cd ../pzdesign && GOWORK=off staticcheck ./cmd/unify)
(cd ../pzdesign && GOWORK=off go test -race ./cmd/unify)
git -C ../pzdesign diff --check
```

CI 使用 Go 1.23.5 和 staticcheck v0.5.1，并检查 push 或 pull request 的已提交差异范围；本地 `git diff --check` 仍用于未提交差异。Docker smoke、数据库支持的管理 UI 测试和 schema drift 是明确的本地/运维门禁，不能因为托管 CI 通过而省略。

## 14. 备份、恢复与值班交接

生产至少备份 MySQL、配置密文来源、上传资产、原始日志、部署清单和版本化二进制。Redis 静态缓存可从 MySQL 重建，但频控、上传人群和短期外部需求方状态是否需要持久化，必须由部署的 Redis 持久化与恢复策略明确决定。NATS Core 日志是尽力传输，不能当作唯一耐久备份。

恢复演练应在非生产环境定期执行，并覆盖：

- 恢复 MySQL 后的 schema/数据检查；
- Redis 全量缓存重建和路由检查；
- 本地 spread 快照重发、节点重载或重启；
- 管理登录、DSP `/bid`、流量方 `/pz`、跟踪和外部需求方回调 smoke；
- 日志时间片、账务重放和报表核对；
- 上一版本二进制回滚。

值班交接至少记录当前发布版本、进行中的 schema/cache 变更、最近缓存和路由生成时间、积压回调数、缺失日志时间片、临时开关、已知受影响流量范围以及下一步负责人。

## 15. 相关资料

- [生产运行手册（详细 systemd 与发布说明）](production-runbook.md)
- [各运维命令完整参数与输出](operational-commands.md)
- [本地 Docker 环境](local-docker-runtime.md)
- [数据库基线与漂移规则](database-baseline.md)
- [Redis、NATS、spread 与进程内缓存](multiple-cache.md)
- [MaxMind 运行资产](maxmind-runtime.md)
- [DSP 工作流](dsp-workflow.md)
- [测量与账务](openrtb-measurement.md)
- [流量方直连接入协议](ssp-direct-traffic.md)
- [外部 DSP / ADX 需求方接入](middleman-adx.md)
- [Genelet 框架运行约定](../../pzdesign/docs/genelet-manual.md)
