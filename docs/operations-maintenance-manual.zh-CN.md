# 系统运维与维护手册

本手册面向 Aofei / Winter DSP 的系统运维人员、值班人员和代码维护者。它把生产部署、日常任务、缓存、日志账务、监控、故障处理和变更验证汇总成一个中文入口。生产环境的主机清单、域名、证书、凭据、备份平台和告警系统仍由具体部署维护。

原有 D/P/R/I/S/A/O 基线截至 A02 已完成；D04、D05 与 S06 已完成，当前评审
整改从 P03 继续到 S05、O03、R03、A03。P03 的威胁契约、默认关闭的 v2 定位符
读取器及 SDK/服务端请求认证已完成，其余执行与灰度任务仍未完成。I02 原生
Android/iOS SDK 仍未实现，且须在 P03、S05
完成并出现具名移动端集成需求后才可启动。D03 已实现但真实外部需求方流量须
单独灰度；I03、S02、S03、A02 均为默认关闭的独立功能。任何生产启用都必须
结合尚未完成的整改项评估风险。O02 定义了单区域目标和证据格式，但在没有命名
生产测量窗口与服务商恢复证据前，不得宣称已达到 99.9% 或生产 RPO/RTO。完整
状态和权威契约见[文档与里程碑索引](README.md)。

## 1. 系统边界

三个相邻 Go 模块共同组成当前服务：

| 代码库 | 责任 |
|---|---|
| `aofei` | DSP/SSP 竞价、匹配、测量、缓存编译、日志与账务作业、MySQL 基线和本地 Docker 环境。 |
| `../pzdesign` | `cmd/unify` HTTP 服务、Summer 管理 UI、模板和静态资源。 |
| `../genelet` | pzdesign 使用的通用 Genelet Web 框架；单独版本化和测试。 |

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
| 转化与实验维护节点 | `action-measurement`、`report-experiment` 定时任务 | 单例或受限维护运行；不得放进竞价请求路径。 |
| 财务维护节点 | `accounting`、`hosted-payment` | 受限人工/定时维护；`hosted-payment` 只能检查健康和保留期，不能自动转账。 |
| 流量质量维护节点 | `traffic-quality` | 受限质量运营主机；只接受有界聚合证据。 |
| 身份维护节点 | `identity-admin` | 受限身份维护主机；用于分析员授权、TOTP 重置和安全审计保留期。 |
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
/opt/aofei/bin/accounting
/opt/aofei/bin/action-measurement
/opt/aofei/bin/report-experiment
/opt/aofei/bin/traffic-quality
/opt/aofei/bin/hosted-payment
/opt/aofei/bin/identity-admin
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
- `action_token_ttl_seconds`、`action_click_window_hours`、`action_view_window_hours`、`action_max_age_hours`、`action_request_skew_seconds`、`action_retention_hours`：转化链路令牌、点击/浏览归因、最晚事件、请求签名时钟窗口和 MySQL 行为/触点保留期；默认分别为 30 天、720 小时、168 小时、2160 小时、300 秒和 2160 小时；保留期不得短于最晚事件窗口；
- `cap_state_ttl_seconds`：用户频控状态空闲 TTL，默认 90 天；更新不会缩短更长的现有 TTL；
- `delivery_cache_max_age_seconds`：投放预算、暂停、时间和时段快照的最大可接受年龄，默认 900 秒；超过后相关旧候选 fail-closed；
- `delivery_reservation_ttl_seconds`：竞价预留元数据寿命，必须覆盖签名回调 TTL 加 5 分钟时钟偏差，默认 86700 秒；
- `delivery_state_ttl_seconds`：每日预算状态跨 UTC 日保留和账务对账的宽限期，默认 172800 秒；必须不短于预留 TTL 加缓存最大年龄；总预算计数保持为持久 Redis 状态；
- `middleman_enabled`：外部 DSP / ADX 需求方竞价总开关；
- `privacy_contextual_middleman_enabled`：对外披露的独立开关，默认关闭；即使本地已获个性化授权，对外请求仍只发送逐竞价方、逐展示位清理后的上下文数据；
- `middleman_always_enabled`：`Always` 路由独立开关，默认关闭；
- `middleman_exchange_domain`、`middleman_timeout_ms`、`middleman_max_bidders_per_imp`：转发身份与预算；
- `middleman_route_cache_ttl_ms`：HTTP 进程内路由快照/错误缓存时间，默认 5 秒；
- `middleman_callback_base_url`、回调 TTL 和回调超时：代理回调的公网地址和生命周期；回调 TTL 必须覆盖跟踪签名有效期、5 分钟未来时钟偏差和 processing 租约，24 小时签名的默认值为 86700 秒；回调超时范围为 1..60000 毫秒；
- `trusted_proxy_cidrs`：仅列出真正受控的反向代理；否则客户端可伪造 IP；
- `privacy_tcf_vendor_id`：W8M 的 TCF Vendor ID；默认 `0` 表示禁用个性化处理，不能凭空填写；
- `privacy_tcf_min_policy_version` 与 `privacy_tcf_purpose_ids`：经法务/政策评审确认的最低 TCF 政策版本和所需目的；
- `privacy_browser_id_ttl_seconds`、`privacy_log_retention_hours`、`privacy_audience_ttl_seconds`：浏览器标识、审计日志和上传人群的默认保留期；
- Summer `ServerURL` 与 `CORSOrigins`：管理 UI 仅允许完全匹配的来源；
- Summer 数据库登录签发器：使用 `Password_hash: "passwd"` 校验 bcrypt 密码，并按查询返回列的顺序完整配置 `OutPars`，其中必须包含 `passwd`。
- Summer `Identity`：默认关闭。启用前先迁移 6 张身份表和 2 个不可变审计
  触发器；由密钥系统向每台 `unify` 节点和受限维护主机提供
  `Identity.KeyEnv` 指定的同一枚 32 字节密钥；逐角色复核 `Permissions`，只读
  分析角色保持 `RequireGrant=true`。身份密钥不能写进 JSON，也不能在不同节点
  各自生成。
- `management_api`：外部广告主管理 API 默认关闭，必须先完成 S02 身份和 I03
  数据库迁移。`key_env` 只填写 32 字节 HMAC 密钥的环境变量名，密钥值不得写进
  JSON；复核每凭证/每广告主分钟配额、请求超时、请求体上限和缓存激活期限。
  管理 API 与竞价流量使用不同的准入计数，Redis 配额不可用时 API 返回 503，
  不能绕过。完整启用、轮换、回滚和审计流程见
  [广告主管理 API 契约](advertiser-management-api.md)。
- `traffic_quality`：可解释的流量质量审查默认关闭。启用前先迁移 9 张 S03 表和
  10 个不可变/保留期触发器，启用 S02 身份并逐项授予 `quality.*` 权限。
  `digest_key_env` 只写至少 32 字节独立 HMAC 密钥的环境变量名，密钥值不得写入
  JSON；`enforcement_refresh_seconds` 默认 30 秒，
  `enforcement_max_age_seconds` 默认 120 秒。启用后若密钥、表或首次执行快照加载
  缺失，服务会拒绝启动；后续刷新失败只保留尚未过期的最后快照，过期后竞价
  fail-open。
- `hosted_payments`：广告主托管付款和流量方收款默认关闭。JSON 只填写
  Stripe API、当前回调密钥和旧回调密钥的三个不同环境变量名，不得填写密钥
  值。优先使用只授予 Customer、Checkout Session、Account/Account Link、
  Transfer、Refund 和 Balance Transaction 所需权限的 `rk_test_`/`rk_live_`
  受限密钥；`sk_test_`/`sk_live_` 仅作为无法使用受限密钥时的后备。启用前必须迁移
  6 张表和 12 个保护触发器、启用 S02 身份、复核
  `payment.*` 权限与双人复核、保留 `/webhooks/stripe` 原始正文并禁止代理缓存/
  验证码挑战。先完成测试模式、回调重放/乱序、退款/争议、故障回退和恢复演练。
  `live_mode` 是单独的法务、财务、税务、风控、隐私和客服上线决定，不能因代码
  已部署而自动开启。详见[托管资金与结算](hosted-funding-payout.md)。

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

从已评审的提交构建。至少先运行三个 Go 模块的全量测试，再安装二进制：

```bash
GOWORK=off go test ./...
GOWORK=off go install ./cmd/accounting ./cmd/action-measurement \
  ./cmd/hosted-payment ./cmd/ledger ./cmd/maxmind \
  ./cmd/mid-callback-retry ./cmd/nats-client ./cmd/redis-cache \
  ./cmd/report-experiment ./cmd/spread ./cmd/traffic-quality

(cd ../pzdesign && \
  GOWORK=off go test ./... && \
  GOWORK=off go install ./cmd/unify ./cmd/identity-admin)

(cd ../genelet && GOWORK=off go test ./...)
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

流量方上线或修改 Web/App、广告位尺寸、状态、最低竞价、供应分类或卖方资料后，先执行只读
商业库存检查：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/redis-cache -validate-publishers
```

该模式只读 MySQL，不连接或修改 Redis，也不取得写锁。它检查活动流量方、
流量源类型与身份、广告位尺寸及非负有限 USD CPM 底价、受控供应分类和卖方
授权状态，并输出确定性的 `site_token`/`slot_token` 清单、令牌版本以及可公开的
请求认证刷新/失效/轮换窗口元数据。命令与 `/pz`、发布商页面使用同一份配置；
启用 v2 时若部署密钥不可用则安全失败，但绝不输出密钥环境变量名/值、发布商
凭据 ID 或私钥。出现任何错误都不能发布该世代。卖方资料改动会撤销已有授权；
运营人员须复核卖方 ID、类型、ASI、公开名称和域名后再授权，不能把透明度资料
当成结算指令。

P01 发布顺序是：先安装新版 `redis-cache`，检查库存，发布并检查完整新
缓存世代，再滚动更新 HTTP 节点。新 HTTP 节点遇到缺少类型/底价的旧
publisher 缓存会安全拒绝 `/pz`。回滚时保留新缓存到 HTTP 回滚完成，或
先完成全部 HTTP 回滚再恢复旧缓存。完整验收、灰度、停用和回滚见
[流量方商业上线与回滚](publisher-activation.md)。

P02 的卖方、流量源和广告位分类使用同一份 publisher Gob 缓存做加法扩展。
旧读取端会忽略新增字段；新读取端遇到兼容的旧世代时把缺失分类记为
`Unknown`，绝不推断卖方已授权。启用外部需求方前，须分别验证已授权自有
流量、已授权代理/转售流量、未授权卖方和客户端伪造 `source` 四种情况；只有
服务端批准状态可以生成 `source.schain`。

D02 上线前还必须审计全部活动广告组的 `cost_type`/`cost` 以及活动素材的
媒体类型、尺寸、权重、来源 URL、MIME、落地页和跟踪地址。运行时只支持
正数 USD CPM，不会把历史 CPC、CPA 或 ROI 用固定系数换算；每条历史记录
都需要业务负责人给出经审核的 CPM，无法确认的记录应先停用。发布时冻结
广告主编辑，先迁移 MySQL 并安装新版单例 `redis-cache`，发布完整 Redis 和
spread 世代，再逐步更新 HTTP 节点，最后启用新版素材管理。旧 HTTP 可读取
新增字段，因此回滚 HTTP 时保留新缓存；不要在仍有新版节点时回灌旧素材
缓存。SQL 审计、素材格式和完整顺序见
[竞价、计价与素材契约](auction-pricing-creatives.md)。

生产缓存节点执行：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/redis-cache -cache=redis
```

带预算、暂停、起止时间或每周时段的候选使用缓存内生成时间执行安全截止。默认 900 秒最大年龄时，完整 `redis-cache -cache=redis`（spread/local 部署使用 `-cache=all`）应由单例 timer 至少每 5 分钟成功完成一次，使正常调度有三次机会并把 MySQL 修改的最坏传播时间限制在 15 分钟内。只运行 `-cache=routes` 不会刷新投放策略。缓存发布持续失败时，过期的本地投放候选会停止竞价；这是防止暂停或预算修改无限期失效的安全行为。

常见模式：

| 模式 | 用途 |
|---|---|
| `-cache=redis` | 重建 Redis 全部静态缓存和外部需求方路由。 |
| `-cache=spread` | 通过 NATS 发布 spread/local 静态快照。 |
| `-cache=all` | 同时执行 spread 和 Redis 发布。 |
| `-cache=routes` | 只刷新外部需求方路由，不改其他缓存族。 |
| `-cache=routes -read` | 只读路由 JSON 与元数据，不取得写锁。 |
| `-validate-publishers` | 只读检查可发布流量资源并输出类型、尺寸、底价、供应分类、卖方授权状态和打包令牌。 |
| `-validate-middleman -activation-stage=preflight` | 只读比较 MySQL 与 Redis v2 路由、检查配置/凭据引用，不输出凭据值。 |

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

`creative` 条目必须带有明确的 Banner/Video/Native 类型；MIME 应由上传记录
提供，或可从素材 URL 的文件扩展名可靠推断。新版 HTTP 遇到缺少类型或 MIME
不可判定的旧条目会拒绝该素材而不是猜测。观察
`aofei_creative_rejections_total`，上线后持续增长通常表示 MySQL 审计或缓存
发布遗漏，应定位并停用具体记录，不能临时放宽素材校验。

`middleman:routes:v2` 是当前优先格式；旧 `middleman:routes` 仅用于滚动发布兼容。管理 UI 中路由组、路由竞价方或目标的编辑只写 MySQL，不会自动刷新 Redis。编辑后必须在单例缓存节点运行完整或 route-only 刷新，并等待各 HTTP 进程的短期路由快照过期。

路由发布后，应在每个灰度节点的真实服务环境运行
`-validate-middleman -activation-stage=preflight`。它会解析环境中的
`credential_ref`，但输出只包含计数、开关、路由高水位和校验和，不包含请求头
值。`fallback` 阶段要求总开关和隐私披露开关开启、Always 关闭；`always`
阶段还要求 Always 独立开关及有效 Always 路由。完整灰度、对账、停用、轮换和
回滚步骤见[外部 DSP / AdX 需求方灰度启用](middleman-activation.md)。

路由刷新失败时，HTTP 进程会在短时间内缓存错误并禁用外部需求方扇出，不会继续使用过期路由授权流量；本地已有赢家仍按正常规则保留。即使取消了发起刷新请求的客户端，共享加载也使用独立超时继续服务其他等待者。

以下 Redis 数据是共享可变状态，静态缓存维护不能误删：

```text
bothcap:<user_id>
upload:<adv_id>:<marker>
middleman:cb:<token>
middleman:click:<request_token>:<imp_id>
middleman:bill:<token>
middleman:notify:<source>:<token>
middleman:publish:<source>:<token>
跟踪事件重复抑制和频控事件标记
delivery:reservation:<token>
delivery:budget:total:<balance_id>
delivery:budget:daily:<UTC日期>:<balance_id>
```

`middleman:notify:*` 记录向下游发送回调的状态和结果，
`middleman:publish:*` 记录本地 NATS 事实是否已发布，两者不能合并或单独批量
删除。这样本地发布失败后的重试不会再次发送已经成功的下游回调。若下游已接收
回调、但完成状态写入失败，重试可能再次发送；这是明确保留的 at-least-once
边界，接入方必须按竞价和广告位身份实现幂等。
notify、publish 与 bill 首先写入带随机 owner 的短期 processing claim，成功后
才按 owner 原子转成完整 callback TTL 的完成标记。失败清理使用脱离 HTTP 取消的
有界上下文；进程退出只会留下短租约，旧 owner 不能删除后续请求取得的新 claim。
`/mid/*` 的 Redis/本地发布依赖失败，以及无法写入持久重试队列的可重试下游失败，
返回 `503` 让交易平台重试；签名错误、过期/缺失/损坏的回调上下文返回 `400`。
已成功写入持久队列的下游失败仍返回正常 `204`，队列任务只重试下游转发。

`bothcap` 更新保证至少保留配置 TTL，并且不会缩短更长 TTL。version 2
频控值保留旧进程可读的 12 字节前缀，并以 UTC epoch-minute 扩展记录权威开始和
最近时间；新进程兼容旧值，旧进程可读取饱和前缀。滚动升级期间不要扫描或删除
`bothcap:*`，被回调更新的旧值会自动升级；回滚时也保留现有值。展示/点击重复
标记的过期时间对应签名的确切有效截止点，最长可能是配置 TTL 再加接收端允许的
5 分钟未来时钟偏差。
本地计量发布完成后，完成标记先于幂等的展示确认、点击计数或 loss 释放写入；
完成标记上的重试不重复发布计量，只重试该投放副作用。带预留 token 的 processing
重复请求返回 `503`，直到 owner 完成或短租约到期，不能清理这些键来绕过重试。
`upload:<adv_id>:<marker>` 在写入成员的同一个 Redis 脚本中设置上传人群 TTL；持久键或较短 TTL 会提升到配置值，较长 TTL 不会被缩短。删除必须按已核验的广告主、marker 和标识符精确执行，不能导出或扫描相邻广告主的数据。

### spread/local 模式

`spread` 将 NATS 缓存消息原子写入：

```text
<spread>/pubmap/
<spread>/audience/
<spread>/creative/
<spread>/slot/<size_id>/
```

DSP 在启动时把这些文件载入内存，热请求不读取文件系统。local 模式会以投放缓存最大年龄（以及更短的 `local_cache_max_age_seconds`，若配置）三分之一为周期自动原子重载文件世代；默认每 5 分钟重载一次。`cmd/spread` 和单例 `redis-cache -cache=all` 仍必须持续运行，自动重载不会自行从 MySQL 生成新文件。外部需求方路由仍是 Redis-only；即使本地静态投放不依赖 Redis，启用外部需求方竞价或预算限制的节点仍需 Redis。

`local_cache_max_age_seconds` 仍是整个本地静态世代的告警阈值；但投放策略另有强制的 `delivery_cache_max_age_seconds`。因此一般静态数据过旧只告警，带 D01 投放策略的候选过期后会 fail-closed。监控告警与投放拒绝指标，并修复 spread/缓存生成链路，不能只反复重启节点加载同一份过期文件。

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
进入 NATS 前，请求会删除原始用户/设备标识、IP、原始 UA、精确位置、同意字符串和未受控扩展；属性审计删除标识和精确派生事实，并记录有限枚举的 `privacy_mode` 与 `privacy_reason`。`nats-client` 在启动和日志轮转时按 `privacy_log_retention_hours` 删除四类已生成的过期文件，不会删除无关文件或符号链接。日志卷和备份仍必须由部署平台加密。

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

每个成功时间片还会用完整 `ledger_adv` 事实对账活动/广告组总预算计数，并用当前 UTC 日期的时间片对账每日计数；每日任务再用 `daily_adv` 校正当日基线。`adv_balance.current_day` 标识每日计数属于哪个 UTC 日期。缓存编译只把当天计数作为每日基线，且只会提高 Redis 已预留值，不会用滞后的账务值降低在途计数。

### 8.3 转化归因维护

`/action` 直接把幂等行为事实写入 MySQL，不经过投放预留或财务账本。展示/
点击 touch 写入失败不会让合法跟踪请求失败，因此应定期重跑仍为“未归因”
的行为：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/action-measurement -action=reconcile -limit=1000
```

按 `action_retention_hours` 生成的到期时间分批清理：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/action-measurement -action=prune -limit=10000
```

两个任务都应由单个运维定时器执行并监控退出状态；多实例不会重复改写已
归因事实，但会造成无意义的数据库竞争。隐私人员完成身份与权限核验后，
可用 `-action=export|delete -pseudonym=<64位小写十六进制值>` 做精确范围的
导出或删除。不要把该假名放入共享命令日志；删除还必须按备份策略登记。

### 8.4 市场分析与对照实验

每个成功账务时间片会在同一事务中写入 `report_delivery`。先部署 R02 schema
迁移，再部署账务写入程序和 Summer 报表；回滚应用时保留报表事实用于对账，
不要直接删表。广告主查询必须带登录会话中的 `adv_id`，流量方查询必须带会话
中的 `pub_id`，角色判断只能使用 Genelet 注入的 `_grole`，不能信任请求参数中
看似有效的 `admin_id`。

报表使用 UTC、USD 六位小数和 `usd-cpm-impression-v2`。`current`、`partial`、
`unavailable`、`unknown` 必须按来源高水位解释；依赖故障不能显示为真实零。
主库保留目标不超过 400 天。连续三个生产复核窗口出现账户报表 p95 大于 2
秒、运营报表 p95 大于 5 秒、保留行数超过 5000 万或要求超过 400 天保留时，
才评估分区、汇总表或 OLAP；迁移前必须保留原权限、口径和对账关系。

运行独立的无线上数据基准：

```bash
./scripts/aofei-reporting-benchmark.sh
```

实验只能由授权 OS 主体显式创建和变更状态：

```bash
AOFEI=/etc/aofei/aofei.json /opt/aofei/bin/report-experiment -action=list
AOFEI=/etc/aofei/aofei.json /opt/aofei/bin/report-experiment \
  -action=start -experiment-id=7 -reason='已批准开始'
```

创建、开始、停止和完成都必须保留审核理由。应用集成按
`LoadExperiment -> Assign -> RecordExposure -> RecordOutcome` 执行，只传 32
字节十六进制假名/幂等摘要；禁止传入账户、Cookie、邮箱、设备或转化原始
标识。结果只允许实验声明的主指标或护栏指标，且不能自动修改出价或预算。
每个实验必须设置 24–9600 小时保留期，默认 2160 小时。单例定时任务运行
`report-experiment -action=prune -limit=1000`，在一个事务中先删到期结果再删
曝光。经核验的删除请求使用精确实验 ID、版本和 64 位 subject hash 执行
`-action=delete-subject`；hash 不得进入共享 shell 历史/日志，审核理由不得含
标识符，审计只记录主体和理由。
完整契约见[市场分析与对照实验](marketplace-analytics-experiments.md)。

### 8.5 外部需求方回调重试

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

### 8.6 账单与人工结算

先确认对应 UTC 日的时间片和每日账务均已完成，再由单例运营节点使用 `accounting` 生成账单。下面以广告主日账单为例：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/accounting -action=create -party=advertiser -party-id=7 \
  -cadence=daily -from=2026-08-01 -to=2026-08-01 \
  -request-key=adv-7-20260801-v1 -reason='UTC daily close'
```

命令从进程的有效 Unix UID 取得不可覆盖的审计身份。制单、确认和结算登记必须分别由不同的非共享 Unix 运营账户执行；创建人不能确认自己的账单，确认人不能登记同一账单的结算。确认前程序会重新读取日账务来源，来源已变化时拒绝确认。`-action=reconcile-middleman` 可按账期只读核对外部需求方的非负收费、应付和差额是否满足“收费减应付等于差额”。确认、暂停、结算、调整、更正、只读对账和 CSV 导出的完整命令见 [账单与人工结算操作契约](accounting-settlement.md)。

结算登记只保存 `invoice:`、`payout:` 或 `manual:` 开头的不透明证据编号。不得把银行卡号、银行账号、路由号码、邮件地址或地址写入证据编号、原因或 CSV。旧版余额充值及银行卡/支票/支付宝/微信支付模块已从活动路由和模板中停用；生产部署迁移前必须清除遗留敏感值，并由合规的托管支付服务商保管所需支付凭据。

### 8.7 托管付款、打款与回调

`unify` 仅在 `hosted_payments.enabled=true` 时注册精确的
`POST /webhooks/stripe`。反向代理必须透传字节完全一致的正文和
`Stripe-Signature`，只允许 HTTPS JSON POST，并对该路径禁用缓存、正文改写、
重定向和交互式挑战。有效回调先完成签名和时间窗验证，再以服务商事件 ID 和
正文 SHA-256 摘要持久去重；无效签名在访问数据库前返回 400，暂时处理失败返回
503 以便服务商重试。出站 API 和回调端点应统一固定为 Stripe
`2024-06-20` 版本，生产回调事件中的 `api_version` 必须完全一致；版本升级必须按
服务商契约变更评审。Connect 收款绑定只有在
资料已提交、打款已启用且 `transfers` 能力为 `active` 时才进入待复核状态。
广告主付款与媒体方打款都必须先有独立复核通过且服务商就绪的绑定。媒体方绑定
只在 W8M 保存不可变的两位国家或地区代码；同一幂等请求号改换代码会被拒绝。
首次提交会把所用绑定 ID 固定到资金操作；即使之后核准了替代绑定，恢复提交也
必须继续使用原客户或收款账户标识，不能改变同一服务商幂等键的请求参数。
Stripe 事件目的地须同时覆盖平台账户事件和 Connect 关联账户事件；关联账户事件
必须携带顶层 `account` 标识。W8M 只用与收款绑定完全一致的关联账户处理
`account.updated` 与 `payout.failed`，关联账户的直连收款等其他事件不能凭元数据
认领平台的广告主付款、退款或媒体方转账操作。
Stripe 不保证事件送达顺序。若受支持事件暂时找不到所引用的操作或收款绑定，W8M
先保存 `Unresolved` 摘要和异常，再返回 503 请求重试；相同事件 ID 只有在正文摘要
完全一致且归属随后可确认时，才能执行一次受触发器保护的
`Unresolved -> Applied|Ignored` 处理转换。事件类型、对象、服务商时间、正文摘要和
接收时间始终不可修改。

只读健康检查不会显示账户或服务商标识：

```bash
AOFEI=/etc/aofei/aofei.json \
  /opt/aofei/bin/hosted-payment -action=health
```

该维护命令只构造数据库只读/保留期能力，不读取 API 或回调密钥值，也不暴露
服务商调用、回调处理或资金操作方法。

对任意“账单已暂停但操作已核准”、超过两分钟的 `Submitting`/`Canceling`、
超过一小时的 `Submitted`、超过配置时限的未决对账项或预期时段内回调量归零告警。
同时监控 `aofei_hosted_payment_webhook_invalid_total`（签名/请求错误）和
`aofei_hosted_payment_webhook_errors_total`（版本、数据库或处理失败）；后者增长时
Stripe 会收到 503 并重试，必须先修复契约或依赖，不能通过清空事件记录消除告警。
`aofei_hosted_payment_webhook_reprocessed_total` 表示乱序依赖补齐后成功处理的
事件；若持续增长，应检查平台与 Connect 事件目的地的延迟和失败投递。
受限维护主机可按保留期分批删除未被对账证据引用的过期事件：

```bash
AOFEI=/etc/aofei/aofei-maintenance.json \
  /opt/aofei/bin/hosted-payment -action=prune-events \
  -actor-admin-id=42 -limit=1000 \
  -reason='经批准的服务商事件保留计划'
```

此命令不能发起付款、打款、退款或对账。资金操作必须从 S02 工作台完成，要求
精确账户权限、近期双重验证、稳定幂等请求号和制单/复核/执行分离。新付款/打款的
`Held` 账单在每次执行前重新读取并拒绝服务商调用；退款则必须精确引用同一广告主
已成功或部分退款的原付款操作，并受父操作累计金额限制，以便账单结算后仍可按 A01
更正流程退款。快速回调先于页面请求返回、客户端取消
或提交后进程退出时，均以相同幂等键恢复，不能另建操作猜测结果。Stripe 不发送
逐笔 `balance_transaction.*` 可用回调；Stripe 可能在 24 小时后清除幂等结果，
因此 W8M 仅允许在首次持久提交后的 23 小时安全窗内按原键恢复。超过安全窗必须
停止服务商调用并人工核对证据，既不能重用旧键，也不能另建新键猜测结果。签名
回调只固化 Charge、Transfer、Refund 及其 `txn_` 关联，对账人员再通过固定版本
API 读取该笔余额交易，并核验
`available` 状态、来源归属、USD 收付方向、金额、手续费和净额。服务商余额、
手续费、净额、退款、争议、拒付、转账冲正和打款失败必须形成明确对账记录，再
由不同人员关闭异常并按 A01 登记结算。故障期间保留所有本地记录，未确认状态
一律视为“待核对”。只要发生过服务商调用，即使本地因网络错误回到 `Approved`，
也不能通过普通取消释放账单容量；应保留原账单与证据，并使用 A01 更正/人工流程
处理从未提交或最终无法恢复的义务。操作原因和幂等请求号不得填写卡号、银行账号
或 `sk_`、`rk_`、`whsec_` 密钥材料。

## 9. 数据库与 MaxMind

### 9.1 数据库生命周期

当前本地模式的活动基线为 `etc/step4_init.sql`。新环境先装入经评审的基线，再执行部署方管理的生产迁移。每次 schema 变更前：

该基线只包含 schema 与非敏感参考目录，不包含广告主、流量方、登录、投放、账务、流量或上传文件记录。本地演示账户和竞价数据统一由完全合成的 `etc/demand.sql` 装入；公开开发密码不得用于生产。

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

在同时承载线上网站的主机上，不得对线上数据库执行 `reset` 或 `reset-sample`。验证时必须为 MySQL、Redis、NATS 使用独立的容器名、卷、端口和数据库名，并设置 `AOFEI_ALLOW_CUSTOM_DESTRUCTIVE=1`。

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
- 启用 `Identity` 后，登录同时校验签名角色 Cookie 和数据库不透明会话；退出
  必须使用 POST+CSRF 页面按钮。TOTP 设置密钥使用 AES-256-GCM 密文，恢复代码
  和会话只保存带域分离的摘要；审计中不得出现密码、验证码、恢复代码、会话
  令牌、请求正文或外部竞价凭据；
- 新密码为 12–128 个 Unicode 字符、最多 72 个 UTF-8 字节且不能含控制字符。
  首次登录按角色要求完成 TOTP 设置并离线保存一次性
  恢复代码；找回密码链接 15 分钟失效，已启用 TOTP 的账户还必须消费一枚恢复
  代码。身份验证器和恢复代码同时丢失时，只能完成线下身份核验后使用受限的
  `identity-admin -action=reset-totp` 并记录原因；
- 只读分析账户只能通过 `identity-admin` 创建，并逐权限、逐资源 grant/revoke。
  `-actor-admin-id` 仅用于审计归属，不能替代维护主机的操作系统身份认证和双人
  复核；新账户密码只能从 `IDENTITY_NEW_PASSWORD` 环境输入，不能作为命令行
  参数；
- 账户安全审计保留 365–2555 天（示例 400 天），只通过有界
  `identity-admin -action=prune-audit` 清理。不得直接更新或删除审计行；
- 管理 API 审计采用相同保留期，但使用独立动作
  `identity-admin -action=prune-api-audit`。命令必须读取维护专用数据库配置，传入
  管理员 ID、单行原因和不超过 10000 的批量上限；HTTP 服务数据库账户不得拥有
  审计删除权限；
- 流量质量规则只能按“草案 → 观察 → 灰度 → 生效”发布。拦截规则进入生效前
  必须有完整证据、灰度命中、人工结论，并且误判率不超过规则上限。跨账户证据
  读取、结案、申诉处理、执行/回滚和账务建议都使用独立 `quality.*` 权限；敏感
  动作要求管理员近期完成 MFA。短期证据只保存带密钥摘要和受控摘要，不得保存
  IP、Cookie、设备标识、竞价 ID、令牌或凭据；
- 定期清理使用受限维护主机上的
  `traffic-quality -action=prune-evidence -actor-admin-id=<id> -limit=1000
  -reason='<单行原因>'`。管理员 ID 仅用于审计归属，不能替代 S02 认证、近期 MFA
  和变更审批。命令清理失败会丢弃无法确认状态的数据库连接；不得直接删除证据、
  判断、案件、计数或审计行；
- 广告主和流量方注册、找回密码使用 Gmail API `users.messages.send`。`Blks._gmail` 设置 `Transport=gmail-api`，JSON 只保存非敏感的可选 `From` 和 `Reply-To`（W8M 使用 `support@w8m.com`）；`GOOGLE_CLIENT_ID`、`GOOGLE_CLIENT_SECRET` 与 `GOOGLE_REFRESH_TOKEN` 必须由权限为 `0600`、仅部署账户可读的环境文件注入。服务会在写库前验证 Google 刷新令牌；缺少 `_gmail`、认证信息不完整或 Google 拒绝令牌时，注册和找回密码返回“邮件服务暂时停用”，登录及已认证工作台不受影响。OAuth 凭据一旦泄漏，应先删除 `_gmail` 并重启服务，再在 Google 端吊销凭据；
- 生产注册和找回密码还必须启用 S06 防滥用边界：Cloudflare Turnstile 仅允许 `w8m.com` 与 `www.w8m.com`，服务端严格核对广告主/流量方注册与找回密码四种固定动作，并在密码散列、Redis、Google、数据库和发信前完成验证；验证通过后再由一条 Redis 原子脚本执行 IP、规范化邮箱和全局限额。Redis 键只保存部署密钥派生的 HMAC 摘要并设置过期时间，转发 IP 只接受 `PUBLIC_ACCOUNT_TRUSTED_PROXY_CIDRS` 明确信任的代理链。Turnstile 密钥、Cloudflare 管理令牌和完整代理列表放入独立 `0600` 环境文件，不得写入 JSON、Git、日志或工单。部署、Cloudflare 边缘限速、监控、轮换及“先停邮件、再停验证”的回滚步骤见 [公开账户防滥用操作契约](public-account-abuse-protection.md)；
- 提交前运行 `./scripts/aofei-public-data-check.sh` 与 `gitleaks git --redact .`，客户 DOCX、数据库/流量快照、运行日志、上传媒体和真实标识符不得进入 Git；
- 管理 UI CORS 只允许 `ServerURL` 和明确列出的精确来源；
- `/pz` 的预检虽然是无凭据宽松 CORS，但实际 `POST` 会在竞价前验证打包令牌和精确站点来源；
- 只信任 `trusted_proxy_cidrs` 内代理提供的转发 IP；
- `/debug/vars` 只允许 `metrics_allowed_cidrs` 中的直连抓取端（默认仅本机），并在 Cloudflare/反向代理再次拒绝公网路径；不得用转发头授权指标访问；
- 静态路径会清理并限制在 `DocumentRoot` 内，上传目录保持非公开；
- 外部需求方回调出口保留 SSRF/DNS 重绑定防护，不以网络便利为由关闭；
- 老旧 SHA1 账户密码上线前全部重置；
- 配置、模板、GeoIP 和二进制使用版本化发布物并校验来源。

## 11. 监控与告警

`unify` 在受保护的 `/debug/vars` 暴露 expvar。公网请求应为 `404`。流量限额、固定维度、容量基线、告警阈值、灰度与回滚详见 [production-traffic-observability.md](production-traffic-observability.md)。至少监控：

| 信号 | 含义与动作 |
|---|---|
| `aofei_traffic_requests_total`、`aofei_traffic_responses_total`、`aofei_traffic_rejections_total`、`aofei_traffic_in_flight` | 区分 ADX/SSP 的正常、无填充、限速、并发过载、超时与体积拒绝；指标不含合作方名称。 |
| `aofei_bid_path_latency_ms` | 固定维度的 count/mean/p50/p95/p99 与桶，包括 ADX、SSP、本地/外部需求、频控、人群、填充/无填充、拒绝/过载。 |
| `aofei_dependency_up`、依赖检查耗时/错误、`aofei_db_pool` | 授权抓取时对 Redis/MySQL 做 100 ms 有界检查并读取 NATS 状态；连续两次不可用应告警。 |
| bid/no-bid 与 SSP 结果 | 按入口、发布版本和流量结构建立基线；突变时检查缓存、MySQL 变更和上游请求。 |
| `aofei_audit_dropped_total`、审计发布错误、队列深度 | 持续增长说明 NATS 或消费端故障，已返回的竞价不会回滚。 |
| `aofei_privacy_decisions_total` | 按有限的模式/原因观察隐私决策；发布或上游集成后比例突变必须调查。 |
| `aofei_privacy_invalid_signals_total` | 无效 GPC/DNT/GDPR/TCF/GPP/US Privacy 信号；不含信号原文，持续增长视为接入事故。 |
| `aofei_privacy_middleman_blocked_total` | 有外部竞价候选但隐私披露开关或决策禁止扇出；确认这是预期保护还是配置遗漏。 |
| `aofei_quality_decisions_total`、`aofei_quality_matched_total`、`aofei_quality_action_*_total` | 固定分类的规则判断、命中和 Observe/Flag/Throttle/Reject/Quarantine 动作；指标不含规则、账户、合作方或事件 ID。 |
| `aofei_quality_dependency_error_total`、`aofei_quality_rollback_total`、执行快照 refresh/error/evaluation 与拦截计数 | 依赖或快照错误属于可用性事故，不得认定为无效流量；使用受限 `traffic-quality -action=health -since-hours=24` 查看逐规则误判上限，超过上限立即停止灰度并回滚。 |
| `aofei_public_account_submissions_total`、`aofei_public_account_turnstile_rejections_total`、`aofei_public_account_rate_limited_total`、`aofei_public_account_dependency_errors_total` | 公开注册/找回密码的固定动作提交、验证拒绝、限速和 Turnstile/Redis/配置错误；持续依赖错误或比例突变应告警，指标不得包含邮箱、IP、账户、主机名或令牌。 |
| `aofei_tracking_replay_redis_errors_total` | 跟踪重复抑制 Redis 失败；合法事件仍 fail-open。 |
| `aofei_tracking_replay_unkeyed_total` | 事件缺少完整重复键；会发布但跳过非幂等频控写入。 |
| `aofei_tracking_cap_update_fail_open_total` | 合法展示/点击已发布，但频控更新失败。 |
| `aofei_tracking_replay_fail_open_total` | 重复控制不可用时接受的事件；注意至少一次计量风险。 |
| `aofei_tracking_retryable_publish_errors_total`、`aofei_tracking_claim_releases_total`、`aofei_tracking_claim_release_errors_total` | 本地计量发布失败后的可重试事实，以及 owner claim 已确认释放或释放失败；与重复抑制指标一起判断重试链路。 |
| `aofei_bothcap_formats_total` | 仅含 `legacy`、`utc_v2`、`malformed` 三种固定键；观察滚动升级进度，`malformed` 增长时停止扩大发布但不得扫描删除用户频控键。 |
| `aofei_middleman_callback_outcomes_total` | 仅含固定的转发/发布重复、可重试、入队和 claim 释放结果；不含 callback token、竞价、用户、合作方端点或广告位标识。 |
| `aofei_action_requests_total`、`aofei_action_accepted_total`、`aofei_action_duplicates_total` | 行为回传总量、新事实和幂等重试；重复突增时检查广告主重试实现。 |
| `aofei_action_rejections_total`、`aofei_action_attributions_total` | 固定原因的拒绝与点击/浏览/未归因结果；签名、过期或未归因比例突变需要调查。 |
| `aofei_action_touches_total`、`aofei_action_touch_errors_total` | MySQL 归因触点写入及失败；失败不影响跟踪响应，但需修复依赖并运行 action reconcile。 |
| `aofei_delivery_reservation_attempts_total`、`aofei_delivery_reservations_total`、`aofei_delivery_reservation_rejected_total` | 预算/节奏原子预留的尝试、成功和硬限制拒绝；与业务流量基线比较。 |
| `aofei_delivery_reservation_errors_total` | Redis 预留不可用；有限额广告 fail-closed，持续增长需要立即处理。 |
| `aofei_delivery_cache_stale_rejected_total`、`aofei_delivery_window_rejected_total`、`aofei_delivery_cached_budget_rejected_total`、`aofei_delivery_policy_errors_total` | 区分过期/未来快照、时间窗口、缓存已耗尽和非法策略；结合缓存任务与活动配置处理。 |
| `aofei_delivery_release_errors_total`、`aofei_delivery_finalize_errors_total`、`aofei_delivery_click_errors_total` | loss/生成失败释放、展示确定和点击计数失败；失败采用保守计数，可能欠投。 |
| `aofei_creative_rejections_total` | 候选素材加载失败，或类型、尺寸、MIME、HTTPS、URL、原生资产不相容；发布后突增时按素材 ID 审计 MySQL 与缓存世代。 |
| `aofei_bothcap_refresh_conflicts_total` | 频控并发冲突；持续上升时检查 Redis 延迟和热点用户。 |
| 外部需求方 route cache hit/miss/refresh/error | refresh error 会暂时禁用扇出，且不会复用过期路由。 |
| 外部需求方回调转发结果与 retry `due` | 检查下游端点、网络、429/5xx 和任务吞吐。 |
| `aofei_local_cache_loaded_at_unix`、`aofei_local_cache_age_seconds`、`aofei_local_cache_stale` | 本地静态缓存年龄；stale 时重载或重启。 |
| `aofei_ssp_policy_rejections_total` | `/pz` 来源策略拒绝；结合 403 访问日志检查站点主机和代理。 |
| `aofei_ssp_inventory_token_outcomes_total` | P03 固定分类的 v1/v2 接受、旧版禁用、非法、混用和未知版本结果；不得把 key、站点或令牌值作为标签。 |
| `aofei_ssp_publisher_auth_outcomes_total` | P03 固定分类的兼容、认证成功、缺失/非法/过期/库存/范围/策略/重放拒绝和依赖错误；不得把凭据、发布商、App、nonce 或签名作为标签。 |
| `aofei_ssp_publisher_auth_snapshot_refreshes_total`、`aofei_ssp_publisher_auth_snapshot_refresh_errors_total`、`aofei_ssp_publisher_auth_snapshot_loaded_at_unix` | P03 SDK 公钥快照刷新和年龄；启用后刷新失败或超过最大年龄会让 SDK 请求以 `503` 关闭。 |

基础设施还应监控 MySQL 连接/慢查询/磁盘、Redis 内存/淘汰/错误/客户端、NATS 可用性/订阅/丢消息、日志磁盘容量、systemd 重启次数、HTTP 延迟和错误率、证书有效期及节点时钟偏差。

## 12. 故障处理

### 12.1 大量无竞价或 no-fill

1. 按 `/bid` 和 `/pz` 区分入口，并确认上游请求形状、域名和尺寸。
2. 检查 `unify` 健康、错误日志和 `/debug/vars`。
3. 检查 `pubmap`/`pubmap:by-id`、`slot:*`、`audience`、`creative` 是否属于当前世代。
4. 检查活动时间、预算、状态、币种、尺寸、ACL、行业和人群定向。
5. 外部需求方流量检查总开关、Always 开关、路由健康、凭据引用、路由缓存生成时间和刷新错误。
6. local/spread 模式检查缓存年龄及节点是否完成重载。

### 12.2 `/pz` 大量 `400`、`401`、`403` 或 `503`

- `400`：检查 JSON、体积、`site`/`slot` 令牌、Web/App 类型、尺寸、非负底价、必填且安全唯一的 `code`、唯一媒体类型和缓存是否更新；
- `403`：检查实际 `Origin`/`Referer` 是否存在且主机与缓存站点完全一致，包括子域差异；
- SDK 可以不带来源，但带了错误来源仍会失败；
- 策略拒绝发生在 Cookie、竞价和审计之前，因此不能只查 NATS 审计。
- 启用 `direct_ssp_auth` 后，`401` 表示 SDK 证明缺失、非法、过期、重放或范围不符；`503` 表示公钥快照或 Redis 重放依赖不可用。不得临时关闭认证来绕过故障。

### 12.3 Redis 故障

- 静态 Redis 模式可能失去 publisher、candidate、audience 和 creative 读取；
- local/spread 模式中不使用频控或上传人群的本地投放可继续，但需要这些共享可变状态的候选会 fail-closed；
- 任何设置了活动或广告组预算上限的本地候选都需要 Redis 原子预留；Redis 不可用时该候选 fail-closed，可按现有规则尝试其他本地候选或外部需求方；
- 合法 `/imp`、`/clk` 的重复 claim 或 cap Redis 错误会 fail-open 并继续发布，计量可能至少一次；
- 外部需求方竞价依赖 Redis 路由和回调上下文，即使本地静态缓存可用也会受影响；
- Redis 恢复或 failover 后重新发布静态缓存，且不要清除仍需保留的频控/上传/回调键族。
- `delivery:budget:total:*` 是防止账务滞后重新开放预算的持久状态。恢复、迁移或人工清理前必须先完成账务对账并确认对应 `adv_balance`；不得用通配符直接删除。
- 预留 token 过期不会自动退回总预算。怀疑因此欠投时，先暂停对应广告组，保留 Redis 与请求/响应/计量证据，运行正常账务任务，再逐项比较 `adv_balance.current_*` 与准确的 `delivery:budget:total:<balance_id>` 键；只有在账务负责人确认差异并批准事故操作后才能修改该准确键，不能仅凭 token 过期推断退款。

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
6. 在灰度节点运行只读 `-validate-middleman`，确认校验和、高水位、开关阶段和
   凭据引用均通过；不要仅凭管理 UI 的健康页启用生产流量。

## 13. 维护变更与验证

代码维护前依次阅读：

1. `memory-bank/product.md`
2. `memory-bank/architecture.md`
3. `memory-bank/tech-stack.md`
4. `memory-bank/milestone.md`
5. 当前里程碑对应的 `memory-bank/status-<lane><number>.md`

使用 Codex 连续执行多个里程碑时，通过
`$memory-bank:memory-bank-goal` 读取根目录 `GOAL.md`。如果存在
`memory-bank/suggested.txt`，它只是一次性启动建议，必须先与当前里程碑和状态
文件核对，启动后或过期时删除。执行时按确认后的 `STATUS_ORDER` 推进，并在每个
里程碑关闭后校正受影响的后续状态文件和剩余依赖顺序。每次里程碑评审从第 1 轮
开始；修复 P1、P2 或更高级别问题后必须重新评审整个里程碑，最多 10 轮。第 10
轮仍有阻断问题时保持里程碑未完成，不得自动进入后续里程碑。

任何改变当前运行配置、schema、缓存契约或运维流程的变更，都必须在同一变更中更新 memory bank 和相关文档。不要重新创建根级产品/架构/状态文档；`backup/` 只保留策略说明，运行快照和第三方数据必须保存在 Git 之外的加密受控存储中。

当前完整验证入口：

```bash
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off staticcheck ./...
GOWORK=off go test -race ./dsp ./match ./internal/jobs/midcallback \
  ./internal/jobs/cache ./internal/jobs/ledger ./internal/jobs/action \
  ./internal/cmdboot ./cmd/spread ./cmd/nats-client ./cmd/action-measurement
GOWORK=off go test ./dsp ./match -run '^$' -bench . -benchmem
./scripts/aofei-doc-check.sh
./scripts/aofei-cache-smoke.sh
./scripts/aofei-recovery-drill.sh
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

单区域至少运行两个相同版本的 `unify` 节点，并由负载均衡器使用不缓存的
`/readyz` 摘除未初始化、正在优雅退出或本地缓存过期的节点；`/healthz`
只表示进程存活。共享 MySQL/Redis/NATS 故障不会通过 readiness 同时摘除
所有节点，其逐功能降级规则、99.9% 指标口径、15 分钟 RPO/60 分钟 RTO
目标、错误预算和恢复顺序见
[单区域可用性、恢复与 SLO](single-region-availability.md)。没有命名的
30 天生产测量窗口，不得对外宣称已经达到 99.9%。

本地可运行 `./scripts/aofei-recovery-drill.sh`：它只创建带随机名称的临时
MySQL/Redis 容器，校验 dump 哈希、schema/routine/trigger、A01 不可变证据、
R01 行为事实并重建 Redis；临时明文 dump 会在退出时删除，因此该脚本不是
生产备份方案或生产 RTO 证明。

值班交接至少记录当前发布版本、进行中的 schema/cache 变更、最近缓存和路由生成时间、积压回调数、缺失日志时间片、临时开关、已知受影响流量范围以及下一步负责人。

## 15. 相关资料

- [文档与里程碑索引](README.md)
- [生产运行手册（详细 systemd 与发布说明）](production-runbook.md)
- [各运维命令完整参数与输出](operational-commands.md)
- [本地 Docker 环境](local-docker-runtime.md)
- [数据库基线与漂移规则](database-baseline.md)
- [Redis、NATS、spread 与进程内缓存](multiple-cache.md)
- [流量方商业上线与回滚](publisher-activation.md)
- [MaxMind 运行资产](maxmind-runtime.md)
- [DSP 工作流](dsp-workflow.md)
- [测量与账务](openrtb-measurement.md)
- [转化与归因](conversion-attribution.md)
- [单区域可用性、恢复与 SLO](single-region-availability.md)
- [隐私、同意与数据治理技术契约](privacy-data-governance.md)
- [身份、双重验证、权限与审计](identity-access-security.md)
- [托管广告主付款与流量方结算](hosted-funding-payout.md)
- [流量质量、反作弊审查与回滚](traffic-quality-anti-fraud.md)
- [流量方直连接入协议](ssp-direct-traffic.md)
- [外部 DSP / ADX 需求方接入](middleman-adx.md)
- [外部 DSP / ADX 需求方灰度启用](middleman-activation.md)
- [Genelet 框架运行约定](../../pzdesign/docs/genelet-manual.md)
