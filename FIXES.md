# XboardNode-Plus 1.20 修复说明

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
- 面板调整节点权限组后，用户接口 ETag 没变化，节点误判用户授权列表未变化。
- 面板或反代缓存返回旧用户授权列表，导致权限组变更后节点仍未拿到新用户。
- 部分精简系统 root 环境没有安装 `sudo`，`xbctl service logs` 无法查看日志。
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

### 4. 修复权限组变更后用户不同步

当面板给某个用户新增节点权限组时，用户自身套餐、UUID、状态可能都没有变化，但“这个节点允许哪些用户连接”的授权列表已经变化。

部分后端在这种权限组变更后不会更新用户接口的 ETag，节点端如果继续发送 `If-None-Match`，可能收到 304 Not Modified，从而误判用户列表没有变化。表现就是：

- 面板已经给用户分配了节点权限。
- 节点端没有把该用户加入 inbound。
- 重启服务后因为重新全量拉取用户，用户立即恢复。

现在用户列表 REST 轮询不再使用 ETag 缓存，每轮都会拉取完整授权用户快照，并显式发送 `Cache-Control: no-cache, no-store`。配置接口仍保留 ETag 优化。

### 5. 修复 xbctl 日志命令的 sudo 兼容性

`xbctl service status`、`xbctl service restart`、`xbctl service logs` 之前会固定调用 `sudo`。在部分精简系统或容器环境里，root 用户已经有权限，但系统没有安装 `sudo`，会出现：

```text
exec: "sudo": executable file not found in $PATH
```

现在 `xbctl` 会先判断当前是否为 root：root 环境直接调用 `systemctl` / `journalctl`，非 root 环境才使用 `sudo`。

### 6. 修复 Xray 用户热更新一致性

如果调用 Xray `AddUser` 或 `UpdateUsers` 失败，节点不会再错误地更新本地内存用户缓存。

处理方式：

- 热更新失败会返回错误。
- 服务层可触发 fallback 或 reload。
- 避免出现“内存认为用户存在，但 Xray 实际没有该用户”的状态。

现场进一步确认，部分环境中 Xray `UserManager` 返回成功后，实际 inbound 用户表仍可能与完整快照不一致，表现为“同步日志成功，但只有重启服务后普通用户恢复”。因此 Xray 用户凭据发生增删或 UUID 变化时，已改为使用完整用户快照重建 Xray 实例，不再依赖 `UserManager` 增量 patch。

### 7. 修复 UUID 变化差异计算

如果同一个用户 ID 的 UUID 发生变化，现在会正确处理为：

- 删除旧 UUID。
- 添加新 UUID。

这可以避免旧 UUID 残留或新 UUID 未写入 inbound 的问题。

### 8. 增加 Xray 用户表自愈重建

现场确认面板已经返回用户可用，且节点 REST 同步也持续成功，但 Xray 仍可能出现实际 inbound 用户表与节点内存用户快照不一致。典型表现是：

- 面板诊断显示用户 `reason=ok`，属于节点权限组且未过期、未超流量、未封禁。
- 节点日志持续显示 `fetched_users` 不变、`added=0 removed=0`、`inbound_user_update=unchanged`。
- 用户连接仍报 `invalid request user id`。
- 重启服务后立即恢复。

现在 Xray 内核会对未变化的完整用户快照进行周期性自愈重建。即使用户数量和哈希没有变化，也会按间隔用当前完整快照重建 Xray 用户表，修复“节点内存认为用户存在，但 Xray 实际 inbound 丢失该用户”的漂移状态。

触发后同步日志会出现：

```text
inbound_user_update=reconciled reload_xray=success
```

该逻辑只针对 `xray` 生效，并带有间隔限制，避免每分钟 REST 轮询都重启内核。

### 9. 增加内核自适应模式

`kernel.type` 支持 `auto`。新安装默认使用自适应模式：

- `xhttp` / `splithttp` 自动使用 `xray`。
- 其他支持的传输协议默认使用 `singbox`。
- 运行日志会同时记录 `configured_kernel` 和 `effective_kernel`，方便确认配置策略和实际运行内核。

### 10. 增加最近访问目标上报

节点端会在用户连接通过内核调度时记录最近访问目标，并在下一次 report 上报给面板。

上报字段包含：

- 用户 ID
- Xray 日志标识，例如 `user@1`
- 来源 IP
- 网络类型
- 访问目标，例如 `tcp:www.google.com:443`
- 访问时间

该功能用于配合面板的“节点同步诊断”插件看板，把节点日志中的 `user@用户ID` 对应到后台真实邮箱，并查看用户最近访问的目标域名/IP。

说明：HTTPS 连接只能看到域名或 IP 与端口，不能看到完整 URL 路径。

### 11. 最近访问改为快速独立上报

1.19 中最近访问目标跟随普通 report 一起上报，如果面板设置的 `server_push_interval` 是 60 秒，看板即使 3 秒刷新，也只能等下一次普通 report 后才看到新访问记录。

1.20 增加独立的 access 上报 ticker：

- 默认每 3 秒上报一次最近访问目标。
- 仅有访问记录时才请求面板，空队列不会发送。
- access-only 上报不会附带 CPU/内存/硬盘状态，避免高频诊断上报覆盖节点负载状态。
- 支持配置 `node.access_report_interval` 调整间隔。

## 验证

已补充并通过相关测试：

- 异常用户快照不会覆盖当前有效用户。
- 配置 304 时仍允许用户同步。
- 用户列表不会因为后端 users ETag 未变化而跳过权限组同步。
- 用户列表请求会显式绕过 HTTP 缓存。
- `xbctl service logs` 在无 sudo 的 root 环境可以正常使用。
- UUID 变化时会同时产生删除和新增差异。
- Xray 用户热更新失败不会错误更新内存状态。
- Xray 用户快照未变化时会按间隔执行完整用户表自愈重建。
- 节点可上报最近访问目标，面板诊断插件可据此显示真实邮箱、来源 IP 和访问目标。
- 最近访问目标独立快速上报，不再等待普通 report 间隔。

本地验证命令：

```bash
go test ./...
```

## 部署建议

升级到 1.20 后，如果再次出现用户无法连接，请优先查看同步日志中的：

- `previous_users`
- `fetched_users`
- `added`
- `removed`
- `inbound_user_update`
- `reload_kernel`
- `reload_xray`
- `inbound_user_update=reconciled`

这些字段可以快速判断是面板返回异常、同步快照异常，还是 Xray inbound 热更新失败。
