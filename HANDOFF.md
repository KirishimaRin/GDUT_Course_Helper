# HANDOFF — 给下一个接手的人（和 AI）

> 记录踩过的坑和关键决策。做新任务前先读这份文件。

## 项目现状（2026-10-25）

**v2.0.0 = mygo 桌面端重构完成**。原 web 版（gin + HTML 模板 + localhost:32555）已整体退役删除，git 历史里还有。

- 技术栈：Go（mygo v0.6.4 桌面框架）+ WebView2 + Vite/TypeScript 前端（无框架，vanilla TS）
- 核心链路：`internal/grabber`（并发抢课，未动）→ `internal/service`（IPC 服务层，替代原 handler）→ `src/mygo.ts`（自动生成的类型化 client）→ 前端视图
- 事件流：grabber 加了 `SetOnResult` / `SetOnStop` 回调 → service 层 `GrabUpdated.Broadcast` → 前端 `events.grabUpdated`。轮询已废除。
- 登录会话只在内存，不持久化（沿用 v1 的安全姿态）。愿望清单和抢课间隔存 localStorage。

## 构建与运行

```bash
bun install
make run     # 开发（Vite 热更新 + Go 热重启）
make build   # 产物 build/windows-amd64/，含便携 exe + Setup 安装包
bun x tsc --noEmit  # 前端类型检查（vite build 不做类型检查！）
```

## 踩过的坑

1. **vite build 不做类型检查**（esbuild 只剥类型）。必须单独跑 `bun x tsc --noEmit`，否则类型错误会静默进产物。
2. **mygo 要求 Go 1.27.3+**。本机 Go 1.26.4 靠 go.mod 的 `go 1.27.3` 指令 + GOTOOLCHAIN=auto 自动下载解决，首次构建多等一会儿。
3. **mygo CLI 在 `~/go/bin/mygo`**，`go install github.com/egoist/mygo/cmd/mygo@latest` 安装。CI 里也要装。
4. **`mygo generate` 的原理是编译 app 并以 `MYGO_GENERATE=1` 跑一次**，所以服务代码必须能在无窗口环境下启动。`src/mygo.ts` 是生成物，已进 .gitignore，不要手改。
5. **`mygo build` 会先重新生成 client 再跑 vite build**，改了 Go 方法签名后直接 build 即可同步到前端。
6. **MenuItem 分隔线是 `mygo.Separator()` 函数**，不是 `{Separator: true}` 字段——编译器会报 unknown field。
7. **`bunx` 不在 Git Bash 的 PATH 里**，用 `bun x` 代替。
8. ** vite 对 side-effect CSS import（`import "./style.css"`）需要 `src/vite-env.d.ts`** 里的 `/// <reference types="vite/client" />`，否则 tsc 报 TS2882。
9. **切换视图（登录 ↔ 主界面）必须 `resetRenders()`** 清掉 onRender 监听，否则旧视图监听器会渲染进已卸载 DOM（getElementById 返回 null 崩掉）。
10. **抢课进行中往清单加课不会立即生效**：`Grabber.StartGrabbing` 启动时拷贝了 courseIDs，下一轮开始才纳入。前端有 toast 提示，后端如要支持动态加课需把目标列表改成共享原子结构。
11. **已知遗留问题（v1 就有，未修）**：`Grabber.results` 的读写用局部 mu 和 g.mutex 混着锁，理论上有数据竞争；GetResults 无锁读。要修就把 results 的访问统一到 g.mutex。
12. **版本号来源是 `mygo.config.ts` 的 version 字段** → `mygo.App.Version()`。发版时改配置文件，不再用 ldflags 注入。
13. **CI 产物路径**：`build/<os>-<arch>/`（windows-amd64 / linux-amd64 / darwin-arm64），release.yml 的 files 按这个路径收集。

## 测试状态

- `go build ./...` ✅、`tsc --noEmit` ✅、`mygo build` ✅（vite 产物 450KB JS / 14.5KB CSS，Go 11.7MB）
- 冒烟测试 ✅：进程启动、窗口创建、WebView2 页面渲染（登录页截图验证）、前端→Go IPC 调用（`App.status()` 成功，版本号正确显示 v2.0.0）
- **未测**：真实 JSESSIONID 登录后的完整链路（拉课程、抢课、退选）——需要选课时间的真实凭据。第一次真机使用时重点看：课程列表加载、抢课事件是否实时出现在日志、抢到课后系统通知。

## 验证桌面端的正确姿势（坑）

1. **这台机器上 `tasklist` 的输出经 Git Bash grep 不可靠**（会漏进程/报错）。查进程一律用 PowerShell：`Get-CimInstance Win32_Process`。
2. **WebView2 子进程（msedgewebview2.exe）在窗口创建后可能延迟几十秒才 spawn**，不能拿它判断成败。这台机器上 SearchHost/Magpie/ishellpro 也有同名子进程，会干扰判断。
3. **验证 UI 用 PrintWindow 截图**（`PW_RENDERFULLCONTENT=2`），不要用 CopyFromScreen（会被用户前台的窗口挡住）也不要 SetForegroundWindow（抢用户焦点且会被 Windows 拒绝）。

## 设计 token

前端所有颜色/圆角都在 `src/style.css` 顶部 `:root`，明暗自动跟随系统（light-dark()）。强调色只有一个变量 `--gh-accent`，改主题就是改它。
