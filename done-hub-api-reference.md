# DoneHub 中转站接口参考文档

本文档整理了 AxonHub 中 DoneHub 类型中转站调用的所有远端接口，用于手动验证接口准确性。

DoneHub 是 one-hub 的衍生项目，API 设计与 new-api 有较大差异。

## 认证方式

DoneHub 使用 **Bearer Token** 认证，所有接口均需在请求头中携带：

```
Authorization: Bearer {token}
```

站点凭据要求：
- 凭据类型：`token`
- 必需字段：`token`（访问令牌）和 `userId`（用户ID）

## 通用响应结构

所有接口响应遵循统一格式：

```json
{
  "success": true,
  "message": "操作成功",
  "data": { ... }
}
```

- `success`：布尔值，表示请求是否成功
- `message`：字符串，请求结果消息
- `data`：响应数据，结构因接口而异

## 分页响应结构

列表类接口的 `data` 字段结构：

```json
{
  "data": [...],
  "page": 1,
  "size": 100,
  "total_count": 250
}
```

- `data`：数据数组
- `page`：当前页码（从 1 开始）
- `size`：每页大小
- `total_count`：总记录数

---

## 一、令牌管理

### 1.1 列出令牌

获取用户的所有令牌（API Key）列表，支持分页。

**请求**

```
GET /api/token?page={page}&size={size}
```

**查询参数**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| page | int | 是 | 页码，从 1 开始 |
| size | int | 是 | 每页大小，建议 100 |

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "success",
  "data": {
    "data": [
      {
        "id": 123,
        "key": "sk-xxx",
        "name": "测试令牌",
        "status": 1,
        "group": "default",
        "remain_quota": 1000000,
        "used_quota": 500000,
        "unlimited_quota": false,
        "expired_time": 1735689600,
        "model_limits_enabled": true,
        "model_limits": "gpt-4,gpt-3.5-turbo"
      }
    ],
    "page": 1,
    "size": 100,
    "total_count": 1
  }
}
```

**响应字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 令牌 ID |
| key | string | API Key 值 |
| name | string | 令牌名称 |
| status | int | 状态：1=启用，2/3/4=禁用 |
| group | string | 所属分组 |
| remain_quota | int | 剩余配额（500000 = 1 USD） |
| used_quota | int | 已用配额 |
| unlimited_quota | bool | 是否无限配额 |
| expired_time | int64 | 过期时间（Unix 时间戳） |
| model_limits_enabled | bool | 是否启用模型限制 |
| model_limits | string | 允许的模型列表（逗号分隔） |

---

### 1.2 获取令牌详情

获取指定令牌的详细信息。

**请求**

```
GET /api/token/{id}
```

**路径参数**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | int | 是 | 令牌 ID |

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "success",
  "data": {
    "id": 123,
    "key": "sk-xxx",
    "name": "测试令牌",
    "status": 1,
    "group": "default",
    "remain_quota": 1000000,
    "used_quota": 500000,
    "unlimited_quota": false,
    "expired_time": 1735689600,
    "model_limits_enabled": true,
    "model_limits": "gpt-4,gpt-3.5-turbo"
  }
}
```

**响应字段说明**

同 1.1 列出令牌。

---

### 1.3 创建令牌

创建新的令牌（API Key）。

**请求**

```
POST /api/token/
```

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "name": "新令牌",
  "status": 1,
  "expired_time": 1735689600,
  "remain_quota": 1000000,
  "unlimited_quota": false,
  "model_limits_enabled": true,
  "model_limits": "gpt-4,gpt-3.5-turbo",
  "allow_ips": "",
  "group": "default"
}
```

**请求字段说明**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| name | string | 是 | 令牌名称 |
| status | int | 否 | 状态：1=启用，2/3/4=禁用 |
| expired_time | int64 | 否 | 过期时间（Unix 时间戳） |
| remain_quota | int | 否 | 剩余配额 |
| unlimited_quota | bool | 否 | 是否无限配额 |
| model_limits_enabled | bool | 否 | 是否启用模型限制 |
| model_limits | string | 否 | 允许的模型列表（逗号分隔） |
| allow_ips | string | 否 | 允许的 IP 地址 |
| group | string | 否 | 所属分组 |

**响应示例**

```json
{
  "success": true,
  "message": "令牌创建成功"
}
```

---

### 1.4 更新令牌

更新已有令牌的配置。

**请求**

```
PUT /api/token/
```

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**

```json
{
  "id": 123,
  "name": "更新后的令牌",
  "status": 1,
  "expired_time": 1735689600,
  "remain_quota": 2000000,
  "unlimited_quota": false,
  "model_limits_enabled": true,
  "model_limits": "gpt-4,gpt-3.5-turbo,claude-3",
  "allow_ips": "192.168.1.1",
  "group": "premium"
}
```

**请求字段说明**

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | int | 是 | 令牌 ID |
| name | string | 是 | 令牌名称 |
| status | int | 否 | 状态：1=启用，2/3/4=禁用 |
| expired_time | int64 | 否 | 过期时间（Unix 时间戳） |
| remain_quota | int | 否 | 剩余配额 |
| unlimited_quota | bool | 否 | 是否无限配额 |
| model_limits_enabled | bool | 否 | 是否启用模型限制 |
| model_limits | string | 否 | 允许的模型列表（逗号分隔） |
| allow_ips | string | 否 | 允许的 IP 地址 |
| group | string | 否 | 所属分组 |

**响应示例**

```json
{
  "success": true,
  "message": "令牌更新成功"
}
```

---

### 1.5 删除令牌

删除指定的令牌。

**请求**

```
DELETE /api/token/{id}
```

**路径参数**

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | int | 是 | 令牌 ID |

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "令牌删除成功"
}
```

---

## 二、分组管理

### 2.1 获取用户分组映射

获取用户可用的分组及其倍率配置。

**请求**

```
GET /api/user_group_map
```

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "success",
  "data": {
    "default": {
      "id": 1,
      "symbol": "default",
      "name": "默认分组",
      "ratio": 1.0
    },
    "premium": {
      "id": 2,
      "symbol": "premium",
      "name": "高级分组",
      "ratio": 0.8
    }
  }
}
```

**响应字段说明**

- `data`：对象，key 为分组 symbol，value 为分组详情
  - `id`：分组 ID
  - `symbol`：分组标识符
  - `name`：分组名称
  - `ratio`：价格倍率

---

## 三、用户信息

### 3.1 获取用户自身信息

获取当前用户的基本信息和余额。

**请求**

```
GET /api/user/self
```

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "success",
  "data": {
    "id": 1,
    "username": "user@example.com",
    "quota": 5000000,
    "used_quota": 2000000,
    "request_count": 1234
  }
}
```

**响应字段说明**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 用户 ID |
| username | string | 用户名 |
| quota | int | 总配额（500000 = 1 USD） |
| used_quota | int | 已用配额 |
| request_count | int | 请求次数 |

**配额换算**

DoneHub 内部配额按 `500000 quota = 1 USD` 换算。

例如：`quota: 5000000` 表示余额 10 USD。

---

## 四、模型价格

### 4.1 获取可用模型价格

获取所有可用模型的价格信息。

**请求**

```
GET /api/available_model
```

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "success",
  "data": {
    "gpt-4": {
      "groups": ["default", "premium"],
      "owned_by": "openai",
      "price": {
        "model": "gpt-4",
        "type": "tokens",
        "channel_type": 1,
        "input": 30.0,
        "output": 60.0,
        "locked": false,
        "extra_ratios": {
          "premium": 0.8
        }
      }
    },
    "gpt-3.5-turbo": {
      "groups": ["default"],
      "owned_by": "openai",
      "price": {
        "model": "gpt-3.5-turbo",
        "type": "tokens",
        "channel_type": 1,
        "input": 0.5,
        "output": 1.5,
        "locked": false
      }
    }
  }
}
```

**响应字段说明**

- `data`：对象，key 为模型名称，value 为模型详情
  - `groups`：可用此模型的分组列表
  - `owned_by`：模型提供商
  - `price`：价格信息
    - `model`：模型名称
    - `type`：计费类型，`tokens` 表示按 token 计费，`times` 表示按次数计费
    - `channel_type`：渠道类型
    - `input`：输入价格（USD/M tokens 或 USD/request）
    - `output`：输出价格（USD/M tokens 或 USD/request）
    - `locked`：是否锁定价格
    - `extra_ratios`：各分组的额外倍率（可选）

**计费单位说明**

- `type: "tokens"`：价格单位为 USD/M tokens（每百万 tokens）
- `type: "times"`：价格单位为 USD/request（每次请求）

---

## 五、站点公告

### 5.1 获取站点公告

获取站点的公告信息。

**请求**

```
GET /api/announcement
```

**请求头**

```
Authorization: Bearer {token}
Content-Type: application/json
```

**响应示例**

```json
{
  "success": true,
  "message": "success",
  "data": {
    "enabled": true,
    "has_more": false,
    "total": 2,
    "items": [
      {
        "id": 1,
        "content": "系统将于本周五进行维护",
        "type": "warning",
        "description": "预计维护时间 2 小时",
        "status": 1,
        "publish_time": 1735689600,
        "created_time": 1735600000,
        "updated_time": 1735600000
      },
      {
        "id": 2,
        "content": "新增 GPT-4 Turbo 模型支持",
        "type": "success",
        "description": "",
        "status": 1,
        "publish_time": 1735603200,
        "created_time": 1735603200,
        "updated_time": 1735603200
      }
    ]
  }
}
```

**响应字段说明**

- `enabled`：布尔值，公告系统是否启用
- `has_more`：布尔值，是否有更多公告
- `total`：公告总数
- `items`：公告列表
  - `id`：公告 ID
  - `content`：公告内容
  - `type`：公告类型，可选值：`default`、`success`、`warning`、`error`、`progress`
  - `description`：公告描述（附加信息）
  - `status`：公告状态
  - `publish_time`：发布时间（Unix 时间戳）
  - `created_time`：创建时间（Unix 时间戳）
  - `updated_time`：更新时间（Unix 时间戳）

**特殊说明**

- 当 `enabled` 为 `false` 时，AxonHub 会将本地公告快照按空列表同步
- 公告时间字段使用 Unix 时间戳格式

---

## 六、签到功能

DoneHub **不支持**内置签到功能。

当调用签到能力时，adapter 返回 `ErrRelaySiteAdapterNotImplemented` 错误。

---

## 附录：错误处理

当接口调用失败时，响应结构为：

```json
{
  "success": false,
  "message": "错误描述信息"
}
```

AxonHub 会解析 `message` 字段并向用户展示错误信息。

---

## 附录：测试建议

### 测试顺序

1. **认证测试**：先测试 GET /api/user/self 验证 token 是否有效
2. **基础查询**：测试 GET /api/token、GET /api/user_group_map、GET /api/available_model
3. **读取详情**：测试 GET /api/token/:id
4. **创建更新**：测试 POST /api/token/、PUT /api/token/
5. **删除操作**：测试 DELETE /api/token/:id（注意备份）
6. **公告系统**：测试 GET /api/announcement

### cURL 示例

```bash
# 1. 获取用户信息
curl -X GET "https://your-donehub.com/api/user/self" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json"

# 2. 列出令牌（第1页，每页100条）
curl -X GET "https://your-donehub.com/api/token?page=1&size=100" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json"

# 3. 创建令牌
curl -X POST "https://your-donehub.com/api/token/" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试令牌",
    "status": 1,
    "group": "default"
  }'
```

---

## 文档维护

- **创建时间**：2026-06-13
- **代码参考**：`internal/server/biz/relay_site_donehub.go`
- **配额换算**：500000 quota = 1 USD
- **分页起始**：从 1 开始
- **认证方式**：Bearer Token
