# ALLinSSL 桌面版

ALLinSSL 桌面版是一款用于证书申请、续期、部署和监控的 Windows 软件，采用 **Tauri 2 + Vue 3 + Go**。当前版本：**v0.0.1**。

Go 业务以静态库链接到桌面程序，页面资源和内置部署插件一并编入 `ALLinSSL.exe`，无需单独启动后端，也不监听本地 HTTP/HTTPS 端口。

## 功能

- 证书申请、续期、导入、导出与管理。
- 可视化工作流与自动化部署，支持工作流批量删除。
- 授权 API 管理，支持批量删除。
- 证书到期监控、监控模板导入导出与通知配置。
- 私有 CA 管理与证书签发。
- Default / Gold 主题，顶部栏跟随主题变化。
- 可选软件开启密码、操作超时锁定与 Windows 原生身份验证重置密码。
- 托盘运行、单实例、原生文件保存与数据目录选择。

云服务操作需要对应平台的有效授权。

## 使用

支持 **Windows 10/11 x64**，需要 Microsoft Edge WebView2 运行时。使用软件无需安装 Node.js、Go 或 Rust。

发行文件提供两种形式：

| 文件 | 用途 |
| --- | --- |
| `ALLinSSL.exe` | 便携版，放入可写目录后直接运行 |
| `ALLinSSL_0.0.1_x64-setup.exe` | NSIS 安装包，缺少 WebView2 时会联网安装运行时 |

WebView2 负责界面渲染，因此系统可能显示它的辅助进程；Go 业务运行在桌面程序内部。应用采用合并的顶部栏，没有浏览器地址栏。

首次启动可以设置软件密码，也可以选择“暂不设置密码”。启用密码后，每次完整启动及设置中的操作超时时间到期后需要解锁。忘记密码时，通过 Windows 原生身份验证确认当前用户后重置，可使用系统密码或已配置的 Windows Hello/PIN。

关闭主窗口会隐藏到托盘，续期和监控继续运行。需要完全退出时，使用托盘菜单的“退出”；软件会停止调度并等待正在执行的任务结束。重复打开会恢复已有窗口。

**更新程序前，请先从托盘退出旧版，再替换程序或运行安装包。**

## 数据保存与备份

默认将数据保存到 `ALLinSSL.exe` 所在目录：

```text
程序目录/
├─ ALLinSSL.exe
├─ allinssl.json    # 数据目录配置，修改目录后生成
├─ data/            # SQLite 数据库、证书和私钥
├─ logs/            # 运行与任务日志
├─ plugins/         # 用户自行安装的额外插件
└─ webview/         # 主题、语言等界面存储
```

在 **设置 → 常用设置 → 数据保存目录** 中选择其他目录，完全退出并重新启动后生效。只切换目录，不自动迁移或删除原数据；空目录会初始化，已有目录会继续读取数据。目录配置始终保存在程序旁的 `allinssl.json`。

程序所在目录和选择的数据目录需要写入权限。移动便携版时，应保留 `allinssl.json`，并确认其指向的数据目录仍可访问。

软件内的“数据备份”仅导出 `data/data.db`。完整备份应在软件完全退出后，复制数据目录中的 `data`、`logs`、`plugins`、`webview`，以及程序旁的 `allinssl.json`（若存在）。数据库、证书、私钥和云平台凭据属于用户数据，请勿提交到 Git 仓库。

## 开发与构建

在 Windows x64 上准备以下工具：

- Node.js 与 pnpm 10.14.0；现有构建使用 Node.js 24.15.0。
- Go 1.24 或更高版本；现有交付构建使用 Go 1.26.8。
- Rust MSVC 工具链、Visual Studio C++ 构建工具与 Windows SDK。
- LLVM-MinGW UCRT x64，用于构建 Go 静态库。
- Microsoft Edge WebView2 运行时。

以下命令在仓库根目录的 PowerShell 中执行。将编译器路径改成实际安装位置：

```powershell
$env:GO_NATIVE_CC = 'C:\工具\llvm-mingw\bin\x86_64-w64-mingw32-clang.exe'
pnpm run setup
pnpm dev
```

`pnpm run setup` 安装根项目和前端工作区依赖，不能简写为 `pnpm setup`。开发模式加载实际构建的前端，修改页面后需重新运行开发命令。

测试与发行构建：

```powershell
pnpm test
pnpm build
```

`pnpm build` 会自动构建前端到 `static/build`，将 Go 编译为 `src-tauri/native/allinssl.lib`，再由 Rust 静态链接并生成 NSIS 安装包。构建不会执行 FTP 或 Git 自动发布。`pnpm test` 运行桌面 Go 生命周期与内置插件的针对性测试，不调用真实云平台。

单独检查 Rust 时，先生成前端和 Go 静态库：

```powershell
pnpm prepare:desktop
cargo test --release --manifest-path src-tauri/Cargo.toml
```

构建产物：

| 路径 | 内容 |
| --- | --- |
| `src-tauri/target/release/allinssl-desktop.exe` | 可单独复制并改名为 `ALLinSSL.exe` 的便携程序 |
| `src-tauri/target/release/bundle/nsis/` | NSIS 安装包 |

Git 仓库仅提交源码、静态资源、构建配置和必要文档。依赖、前端产物、Go 静态库、可执行文件、安装包、压缩包及用户数据不提交；可执行文件和安装包可通过上述命令自行构建。

## 项目结构

```text
backend/        Go 业务、桌面生命周期与调度
cmd/native/     Go 静态库入口
frontend/       Vue 页面及必要的前端工作区包
plugins/        内置插件源码
scripts/        桌面构建与测试脚本
src-tauri/      Tauri 窗口、系统交互与 Go 桥接
static/         监控模板；build 子目录由构建生成
```

根目录的 `plugins/` 为源码，运行时额外插件位于所选数据目录中，请勿将用户插件文件混入源码目录。

## 当前限制

- 首版以 Windows x64 为目标，Windows 11 与其他机器的兼容性仍需实机验收。
- 软件完全退出、电脑关机或休眠时，续期和监控不会持续执行。
- 部分部署命令依赖 Linux 环境，受 Windows 平台限制。
- 当前不提供自动更新、开机启动或后台系统服务，安装包尚未签名。
- 真实证书申请、云部署和通知发送需有效授权；本地测试不代表这些操作已验收。

## 许可证

本项目采用 [AGPL-3.0 许可证](LICENSE)。依赖组件遵循各自的许可证。
