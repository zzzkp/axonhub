# 中转站实现说明

本文说明 AxonHub 中转站模块的实现结构和维护边界。中转站当前支持 `new-api`、`sub2api` 和 `done-hub` 类型站点，用于管理外部中转站后台资源，不直接参与模型请求转发链路。

## 模块边界

中转站模块负责保存外部站点配置、保护访问凭据、调用远端管理接口、同步远端 API Key、分组、余额、模型数据和站点公告，并记录签到和同步结果。

中转站模块不直接替代 `Channel`，也不在请求期间实时依赖远端中转站后台接口。与 `Channel` 的关系只通过显式导入动作建立：用户选择某个远端 API Key 快照后，由中转站服务获取明文 key，并调用现有 `ChannelService.CreateChannel` 创建普通渠道。

相关使用说明见 [中转站指南](../guides/relay-sites.md)。

## 数据模型

中转站使用独立 Ent schema，避免把远端资源状态混入现有渠道模型。

| 实体 | 作用 |
|------|------|
| `RelaySite` | 站点主表，保存名称、类型、Base URL、状态、备注、最近同步状态和自动签到开关 |
| `RelaySiteCredential` | 站点凭据，保存认证方式和 token、账号密码、JWT 等敏感字段 |
| `RelaySiteAPIKey` | 远端 API Key 快照，保存 key 标识、名称、状态、额度、所属分组等 |
| `RelaySiteGroup` | 远端 group 快照，保存 group 名称和相关配置 |
| `RelaySiteBalanceSnapshot` | 余额快照，保存余额、单位、原始 quota 和拉取时间 |
| `RelaySiteModelPrice` | 模型价格快照，保存模型名、输入价格、输出价格和计费单位 |
| `RelaySiteCheckinLog` | 签到记录，保存执行时间、结果和错误信息 |
| `RelaySiteAnnouncement` | 站点公告快照，保存远端公告 ID、内容、类型、发布时间、获取时间和已读时间 |

主要代码路径：

- `internal/ent/schema/relay_site*.go`
- `internal/objects/relay_site.go`
- `internal/ent/relaysite*`

## 后端服务

后端核心服务位于 `internal/server/biz/relay_site.go`，通过 FX 在 `internal/server/biz/fx_module.go` 注册。

`RelaySiteService` 负责：

- 创建、更新、删除站点配置。
- 单独处理凭据写入和保留逻辑。
- 调用 adapter 同步远端资源。
- 将 API Key、group、余额、模型价格和公告写入本地快照。
- 记录同步和签到结果。
- 执行自动签到任务。
- 将远端 API Key 显式导入为 Channel。
- 同步完成后自动将模型价格换算并写入关联渠道。

公告同步会根据远端公告 ID upsert 本地快照。新公告默认未读；当同一公告的内容、类型、附加信息或发布时间发生变化时，会清空 `readAt` 使其重新进入未读状态；远端接口不再返回的公告会从本地删除。

同步流程中，远端接口调用在数据库事务外执行；拿到远端结果后，再在事务内 upsert 本地快照。这样可以避免外部网络调用长时间占用事务。

## Adapter 设计

统一适配器接口位于 `internal/server/biz/relay_site_adapter.go`。站点类型通过 adapter factory 路由到具体实现，当前实现包括 `new-api`、`sub2api` 和 `done-hub`。

adapter 负责屏蔽不同中转站后台接口差异，向 service 提供统一能力：

```text
RelaySiteAdapter
  - ListAPIKeys
  - CreateAPIKey
  - UpdateAPIKey
  - DeleteAPIKey
  - GetAPIKey
  - ListGroups
  - GetBalance
  - ListModelPrices
  - Checkin
  - ListAnnouncements
```

`new-api` 适配实现位于 `internal/server/biz/relay_site_newapi.go`。当前调用的远端接口包括：

- `POST /api/user/login`
- `GET /api/token/`
- `POST /api/token/`
- `PUT /api/token/:id`
- `DELETE /api/token/:id`
- `POST /api/token/:id/key`
- `GET /api/user/self/groups`
- `GET /api/user/self`
- `GET /api/pricing`
- `POST /api/user/checkin`
- `GET /api/status`

访问令牌认证会携带 `Authorization` 和 `New-Api-User`；账号密码认证会先登录获取 session cookie。若远端启用 Turnstile 等登录风控，账号密码登录仍可能被远端策略拦截。

new-api 返回的内部 quota 按 `500000 quota = 1 USD` 换算为 USD 展示，同时保留原始 quota 便于核对。

公告通过 `GET /api/status` 获取，该接口不携带站点凭据。适配器读取 `announcements_enabled` 和 `announcements` 字段；当 `announcements_enabled` 为 `false` 时，本地公告快照会按空列表同步。公告发布时间使用远端 `publishDate` 字段解析为 UTC 时间。

`sub2api` 适配实现位于 `internal/server/biz/relay_site_sub2api.go`。当前调用的远端接口包括：

- `GET /api/v1/keys`
- `GET /api/v1/keys/:id`
- `POST /api/v1/keys`
- `PUT /api/v1/keys/:id`
- `DELETE /api/v1/keys/:id`
- `GET /api/v1/groups/available`
- `GET /api/v1/groups/rates`
- `GET /api/v1/auth/me`
- `GET /api/v1/user/profile`
- `GET /api/v1/announcements`
- `GET /v1/models`

sub2api 使用 dashboard JWT 凭据，站点凭据类型为 `jwt`。`token` 保存 access token，`refreshToken` 和 `tokenExpiresAt` 作为后续刷新能力的预留字段，当前不会自动刷新 access token。

sub2api 的远端 API Key 使用数字 `group_id`，本地快照会把它映射为分组名称，避免前端分组筛选和导入 Channel 时出现 ID 与名称不一致。创建或更新远端 API Key 时，adapter 会把表单中的分组名称解析回远端 `group_id` 后提交。

部分 sub2api 站点不提供 pricing 接口。sub2api adapter 不依赖 pricing 接口同步模型数据，而是使用已同步到的远端 API Key 调用 OpenAI 兼容的 `GET /v1/models`，并按该 key 所属分组写入模型快照的 `enable_groups`。单个 key 的模型接口失败会被跳过，不会导致整个站点同步失败。

sub2api 当前不支持内置签到；调用签到能力时返回”不支持”。模型快照主要用于模型选择和 Channel 导入，不包含 new-api 那类价格倍率数据。

`done-hub` 适配实现位于 `internal/server/biz/relay_site_donehub.go`。done-hub 是 one-hub 的衍生项目，API 设计与 new-api 有较大差异。当前调用的远端接口包括：

- `GET /api/token` - 列出令牌
- `GET /api/token/:id` - 获取令牌详情
- `POST /api/token/` - 创建令牌
- `PUT /api/token/` - 更新令牌
- `DELETE /api/token/:id` - 删除令牌
- `GET /api/group/` - 列出分组
- `GET /api/user_group_map` - 获取分组倍率
- `GET /api/user/self` - 获取用户信息和余额
- `GET /api/available_model` - 获取模型价格
- `GET /api/announcement` - 获取站点公告

done-hub 使用 token 认证，凭据类型为 `token`，需提供 `token` 和 `userId` 字段。认证只需携带 `Authorization: Bearer {token}`，无需额外 header。

done-hub 关键特性：
- 分页从 1 开始，使用 `page` + `size` 参数
- 响应结构为 `{success, data: {data[], page, size, total_count}}`
- 令牌端点为 `/api/token` 而非 `/api/channel/`
- 支持完整的模型价格信息（输入/输出价格 + tokens/times 计费类型）
- 支持丰富的公告系统（5 种公告类型：default、success、warning、error、progress）
- 不支持内置签到功能，调用签到能力时返回”不支持”
- quota 按 `500000 quota = 1 USD` 换算为 USD 展示

done-hub 的模型价格通过 `/api/available_model` 获取，返回每个模型的输入价格（input）、输出价格（output）和计费类型（type: tokens/times）。价格单位为 USD/M tokens（tokens 类型）或 USD/request（times 类型）。

done-hub 的公告系统支持开关控制（enabled 字段）和多种公告类型，公告时间使用 Unix 时间戳（publish_time）。当 `enabled` 为 `false` 时，本地公告快照会按空列表同步。

## GraphQL 接口

GraphQL schema 和 resolver 位于：

- `internal/server/gql/relay_site.graphql`
- `internal/server/gql/relay_site.resolvers.go`
- `internal/server/gql/ent.resolvers.go`
- `internal/server/gql/generated.go`

中转站使用专用 mutation 管理站点和凭据，避免通过 Ent 通用输入暴露敏感凭据字段。当前接口覆盖：

- 创建、更新、删除站点。
- 同步站点资源。
- 手动签到。
- 刷新站点公告。
- 将站点公告标记为已读。
- 创建、更新、删除远端 API Key。
- 为所有 group 创建远端 API Key。
- 导入远端 API Key 为 Channel。

会调用外部网络接口的 mutation 不应被 GraphQL 全局事务包装，避免外部调用期间持有数据库事务。

## 前端结构

前端页面位于 `frontend/src/features/relay-sites` 和 `frontend/src/routes/_authenticated/relay-sites/index.tsx`。

页面入口通过以下文件接入：

- `frontend/src/sidebar.ts`
- `frontend/src/config/route-permission.ts`
- `frontend/src/locales/zh-CN/relaySites.json`
- `frontend/src/locales/en/relaySites.json`

前端功能包括站点分页列表、搜索和状态筛选、新增和编辑站点、手动同步、手动签到、资源快照展示、API Key 管理弹窗、模型与令牌弹窗、签到记录弹窗、公告弹窗，以及导入 Channel 弹窗。

模型与令牌弹窗会按分组展示远端 API Key 和可用模型。令牌列表中的“渠道状态”开关复用显式导入流程：未关联 Channel 时创建普通 Channel，已关联但禁用时重新启用，已启用时禁用对应 Channel。令牌列表中的“端点配置”开关复用现有 `saveChannelEndpoints` mutation，为该令牌关联的 Channel 快速补充或移除消息协议端点。

公告入口位于中转站列表操作菜单。打开公告弹窗时会调用 `refreshRelaySiteAnnouncements` 拉取远端 `/api/status` 并刷新本地快照；弹窗内展示公告内容、类型、发布时间、获取时间和已读状态，并可通过 `markRelaySiteAnnouncementsRead` 将当前站点公告标记为已读。列表的站点列通过 `hasUnreadAnnouncements` 显示未读提示。

## 备份与迁移

中转站页面提供导出和导入备份能力，用于跨实例迁移或本地备份中转站模块数据。

导出接口会生成 JSON 文件，包含站点配置、敏感凭据、API Key 快照、分组、余额快照、模型价格、签到记录和公告快照。由于凭据以可迁移形式写入备份文件，备份文件应按敏感数据保管。

导入接口会在事务内清空当前中转站模块数据，再按备份文件重建中转站、凭据和资源快照。清空过程不复用 `DeleteSite` 业务删除逻辑，因此不会删除由中转站导入生成的现有 `Channel`；这些 Channel 会保留在渠道模块中，但其中的 `relay-site:<id>`、`relay-site-api-key:<id>` 标签仍指向导入前的旧本地 ID，导入后不保证继续作为中转站快捷关联使用。

## 自动签到

站点级自动签到开关保存在 `RelaySite` 上。后台任务每天执行一次，只处理启用且开启自动签到的站点，并逐站点复用 `CheckinSite` 记录成功或失败结果。

自动签到不进入请求转发链路，也不影响普通模型请求。

## Channel 导入

导入 Channel 是显式动作，不做自动同步。流程为：

1. 用户选择远端 API Key 快照。
2. service 通过 adapter 获取远端明文 key。
3. 用户填写 Channel 类型、Base URL、支持模型、默认测试模型等字段。
4. service 调用现有 `ChannelService.CreateChannel` 创建普通渠道。

导入后，中转站和 Channel 没有强绑定关系；后续请求转发仍由现有 Channel、模型关联和负载均衡逻辑处理。前端会通过导入时写入的 tags 查找中转站 API Key 与 Channel 的关联，用于模型与令牌弹窗中的渠道状态和端点配置快捷操作。

端点配置开关只管理消息协议端点，不修改 Channel 默认端点，也不修改请求转发协议选择逻辑。当前自动管理的端点为：

- `openai/responses`
- `anthropic/messages`
- `gemini/contents`

开启端点配置时，前端会保留现有自定义 endpoints，并补齐缺失的上述消息协议 endpoints。关闭端点配置时，只删除 apiFormat 属于上述集合且 `path`、`baseURL`、`transport` 均为空的 endpoint；如果用户在渠道端点配置中为这些协议设置了自定义 path、baseURL 或 transport，则关闭开关不会删除该 endpoint。

## 价格自动同步到渠道

站点同步完成后，系统会自动将模型价格快照换算并写入所有关联渠道的 `ChannelModelPrice`。该功能为衍生动作：换算或写入失败不影响站点同步本身成功，失败信息记录在日志中。

核心换算逻辑位于 `internal/server/biz/relay_site_price_sync.go`。

### 换算规则

渠道价格 `UsagePerUnit` 语义为 **USD 每百万 token**，`FlatFee` 为 **USD 每次请求**。

**new-api** 站点（基于 `500000 quota = 1 USD` 换算基准）：

- 按量计费模型（`quota_type ≠ 1`，BillingUnit = `new-api-ratio`）：
  - prompt price = `model_ratio × group_ratio × 2` USD/M tokens
  - completion price = `completion_ratio × group_ratio × 2` USD/M tokens
- 按次计费模型（`quota_type = 1`，BillingUnit = `new-api-model-price`）：
  - flat fee = `model_price × group_ratio` USD/request

**done-hub** 站点（价格已为 USD/M tokens 语义）：

- tokens 类型（BillingUnit = `done-hub-tokens`）：
  - prompt price = `input × group_ratio` USD/M tokens
  - completion price = `output × group_ratio` USD/M tokens
- times 类型（BillingUnit = `done-hub-times`）：
  - flat fee = `input × group_ratio` USD/request

**sub2api** 站点：模型快照不包含价格数据（`PromptPrice`/`CompletionPrice` 均为 nil），换算跳过该站点的所有关联渠道。

### 分组倍率与模型过滤

换算时从渠道 tags 中解析 `relay-site-api-key:<id>` 标签，查询对应 `RelaySiteAPIKey.GroupName`，再从 `RelaySiteGroup.Ratio` 获取该分组的价格倍率（缺省为 1）。

模型价格快照包含 `enable_groups` 字段（保存在 `price.Raw["enable_groups"]`）。如果该字段为空列表，表示模型对所有分组启用；否则只对列表中的分组启用。换算时会过滤掉对当前渠道所属分组不可用的模型。

### 写入策略

使用 `ChannelService.SaveChannelModelPrices` 全量写入：

- 存在于快照但不在渠道中的模型 → 创建新价格
- 快照与渠道价格不一致 → 更新价格并归档旧版本到 `ChannelModelPriceVersion`
- 存在于渠道但不在快照中的模型 → 删除价格并归档版本

该策略等效"清空后重建"，但额外保留了历史版本用于审计和账单引用。

当换算后某渠道无任何可用价格时（如 sub2api 渠道或所有模型都不对该分组启用），会跳过该渠道而非清空其价格，避免误删手动设置的价格。

### 触发时机

价格同步在以下操作完成后自动触发：

- 手动点击"同步"按钮
- 创建、更新、删除远端 API Key
- 自动签到任务（如果站点启用自动签到）

### 相关代码

- `internal/server/biz/relay_site_price_sync.go` — 换算与同步逻辑
- `internal/server/biz/relay_site.go` 的 `SyncSite` 方法 — 调用入口
- `internal/server/biz/channel_price.go` 的 `SaveChannelModelPrices` — 渠道价格写入

## 扩展新站点类型

新增站点类型时，应优先复用现有边界：

- 保持 `RelaySite` 系列 Ent 模型稳定。
- 在 adapter factory 中新增站点类型分发，当前 `new-api` 对应内部类型值为 `new_api`，`sub2api` 对应内部类型值为 `sub2api`，`done-hub` 对应内部类型值为 `done_hub`。
- 新增具体 adapter 实现远端接口差异。
- 仅在确有差异时扩展凭据或快照字段。
- 不把中转站逻辑接入 `llm`、orchestrator、request 或 Channel 请求转发链路。

## 相关文档

- [中转站指南](../guides/relay-sites.md)
- [渠道管理](../guides/channel-management.md)
- [请求处理流程](../getting-started/request-processing.md)
