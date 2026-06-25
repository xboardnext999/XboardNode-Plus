# XboardNode-Plus 1.14 修复说明

本文记录 XboardNode-Plus 针对运行中用户同步异常的修复内容。

## 修复的问题

部分节点运行一段时间后，已经存在的普通用户会突然无法连接，但管理员或部分用户仍可连接。节点日志可能出现：

```text
proxy/vless/encoding: invalid request user id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

这些用户并不是新创建用户，也没有购买新套餐、套餐过期、流量超限、UUID 变更或被禁用。重启节点后，全量快照重新加载，普通用户恢复连接。

## 根因方向

该问题不是单纯的新增用户增量同步问题，而是运行中的同步或热更新可能导致以下状态不一致：

- REST 同步拿到空响应、失败响应或异常用户快照后，覆盖了当前有效用户列表。
- 配置未变化返回 304 时，用户同步流程可能没有被正确处理。
- Xray inbound 热更新失败，但内存里的用户缓存被错误更新为“已经成功”。
- 同一个用户 ID 的 UUID 变化时，差异计算没有同时执行删除旧 UUID 和添加新 UUID。

这些情况都会导致节点内存、面板用户列表和 Xray 实际 inbound 用户列表不一致，最终表现为老用户 UUID 在 Xray 中找不到。

## 已修复内容

### 1. 同步失败不再覆盖旧用户

当 REST 请求失败、返回空用户列表、JSON 解析失败或用户数量明显异常时，节点不会用异常结果覆盖当前用户列表。

处理方式：

- 保留上一份有效用户快照。
- 记录错误日志。
- 等待下一轮同步重试。

### 2. 增加同步保护和诊断日志

每次同步都会记录关键诊断信息：

```text
sync start
previous users: 201
fetched users: 201
added: 0
removed: 0
inbound user update: success
reload xray: success
```

日志字段包含：

- previous user count
- fetched user count
- added user count
- removed user count
- inbound user update result
- kernel reload result

### 3. 修复配置 304 对用户同步的影响

配置接口返回 304 Not Modified 时，只代表节点配置未变化，不代表用户列表没有变化。

现在配置未变化不会阻断用户列表同步。

### 4. 修复 Xray 用户热更新一致性

如果调用 Xray `AddUser` 或 `UpdateUsers` 失败，节点不会再错误地更新本地内存用户缓存。

处理方式：

- 热更新失败会返回错误。
- 服务层可触发 fallback 或 reload。
- 避免出现“内存认为用户存在，但 Xray 实际没有该用户”的状态。

### 5. 修复 UUID 变化差异计算

如果同一个用户 ID 的 UUID 发生变化，现在会正确处理为：

- 删除旧 UUID。
- 添加新 UUID。

这可以避免旧 UUID 残留或新 UUID 未写入 inbound 的问题。

### 6. 增加内核自适应模式

`kernel.type` 支持 `auto`。新安装默认使用自适应模式：

- `xhttp` / `splithttp` 自动使用 `xray`。
- 其他支持的传输协议默认使用 `singbox`。
- 运行日志会同时记录 `configured_kernel` 和 `effective_kernel`，方便确认配置策略和实际运行内核。

## 验证

已补充并通过相关测试：

- 异常用户快照不会覆盖当前有效用户。
- 配置 304 时仍允许用户同步。
- UUID 变化时会同时产生删除和新增差异。
- Xray 用户热更新失败不会错误更新内存状态。

本地验证命令：

```bash
go test ./...
```

## 部署建议

升级到 1.14 后，如果再次出现用户无法连接，请优先查看同步日志中的：

- `previous_users`
- `fetched_users`
- `added`
- `removed`
- `inbound_user_update`
- `reload_kernel`
- `reload_xray`

这些字段可以快速判断是面板返回异常、同步快照异常，还是 Xray inbound 热更新失败。
