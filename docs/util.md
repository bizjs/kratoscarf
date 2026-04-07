# util — 工具包

通用工具函数集合。每个子包独立，无内部依赖。

## util/id — ID 生成

```
util/id/
└── id.go    # UUID, UUIDv7, ULID, Short, ShortN
```

### 函数一览

| 函数 | 返回值 | 说明 |
|------|--------|------|
| `UUID()` | `string` | UUID v4，随机 |
| `UUIDv7()` | `string` | UUID v7，时间排序 |
| `ULID()` | `string` | ULID，时间排序 + 单调递增 |
| `Short()` | `string` | 12 位 base62 随机 ID |
| `ShortN(n)` | `string` | n 位 base62 随机 ID |

### 使用

```go
import "github.com/bizjs/kratoscarf/util/id"

id.UUID()     // "f47ac10b-58cc-4372-a567-0e02b2c3d479"
id.UUIDv7()   // "018f3b5c-..." 时间排序，适合做主键
id.ULID()     // "01HQJX5P6R..." 时间排序 + 单调递增
id.Short()    // "a3Bx9kLm2Wnp" 12位，URL-safe
id.ShortN(8)  // "Km2x9aLp" 自定义长度
```

### 选型建议

| 场景 | 推荐 | 原因 |
|------|------|------|
| 数据库主键 | `UUIDv7` / `ULID` | 时间排序，B-tree 友好 |
| 分布式 trace ID | `ULID` | 单调递增，无冲突 |
| 短链 / 邀请码 | `Short` / `ShortN` | URL-safe，可控长度 |
| 兼容已有 UUID 系统 | `UUID` | 标准 v4 |

### 并发安全

所有函数线程安全。`ULID()` 内部使用 `sync.Mutex` 保护单调熵源。

---

## util/crypto — 密码学工具

```
util/crypto/
├── cipher.go    # AES-GCM 加解密
└── hash.go      # SHA-256, HMAC-SHA256, Bcrypt
```

### 函数一览

**加密 / 解密**

| 函数 | 说明 |
|------|------|
| `AESKey(bits)` | 生成随机 AES 密钥（128/192/256 位） |
| `AESGCMEncrypt(key, plaintext)` | AES-GCM 加密，返回 `[]byte` |
| `AESGCMDecrypt(key, ciphertext)` | AES-GCM 解密，返回 `[]byte` |
| `AESGCMEncryptString(key, plaintext)` | 加密字符串，返回 base64 |
| `AESGCMDecryptString(key, ciphertext)` | 解密 base64 字符串 |

**哈希 / 签名**

| 函数 | 说明 |
|------|------|
| `SHA256(data)` | SHA-256 哈希，返回 hex |
| `HmacSHA256Key()` | 生成 32 字节随机 HMAC 密钥 |
| `HmacSHA256(key, data)` | HMAC-SHA256 签名，返回 hex |
| `BcryptHash(password)` | Bcrypt 哈希，默认 cost 10 |
| `BcryptHashWithCost(password, cost)` | Bcrypt 哈希，自定义 cost |
| `BcryptVerify(hashed, password)` | Bcrypt 验证，匹配返回 nil |

### 使用 — 密码哈希

```go
import "github.com/bizjs/kratoscarf/util/crypto"

// 注册时哈希密码
hashed, err := crypto.BcryptHash("user-password")
// 存储 hashed 到数据库

// 登录时验证
err := crypto.BcryptVerify(hashed, "user-password")
if err != nil {
    // 密码不匹配
}
```

### 使用 — AES-GCM 加解密

```go
// 生成密钥（一次，安全存储）
key, err := crypto.AESKey(256)

// 加密
encrypted, err := crypto.AESGCMEncryptString(key, "sensitive data")
// encrypted = "base64..." 可安全存储或传输

// 解密
plaintext, err := crypto.AESGCMDecryptString(key, encrypted)
// plaintext = "sensitive data"
```

### 使用 — HMAC 签名验证

```go
// Webhook 签名验证
key, _ := crypto.HmacSHA256Key()
signature := crypto.HmacSHA256(key, []byte(requestBody))

// 验证时重新计算并比较
expected := crypto.HmacSHA256(key, []byte(requestBody))
if signature != expected {
    // 签名不匹配
}
```

### 使用 — 数据指纹

```go
fingerprint := crypto.SHA256([]byte(content))
// "e3b0c44298fc1c149afbf4c8996fb924..." 用于去重、缓存键等
```

### 选型建议

| 场景 | 推荐 | 原因 |
|------|------|------|
| 用户密码存储 | `BcryptHash` | 自适应 cost，抗暴力破解 |
| 敏感数据加密 | `AESGCMEncrypt` | 认证加密，防篡改 |
| Webhook / API 签名 | `HmacSHA256` | 标准 HMAC，可验证 |
| 数据指纹 / 缓存键 | `SHA256` | 快速，确定性 |
