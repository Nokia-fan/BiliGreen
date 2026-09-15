# BiliGreen 0.5.0

[English](README.en.md) · 简体中文

![BiliGreen 应用预览](docs/assets/preview.svg)

[![Release](https://img.shields.io/github/v/release/Nokia-fan/BiliGreen?label=最新版)](https://github.com/Nokia-fan/BiliGreen/releases/latest)
[![Build](https://github.com/Nokia-fan/BiliGreen/actions/workflows/release.yml/badge.svg)](https://github.com/Nokia-fan/BiliGreen/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/Nokia-fan/BiliGreen)](LICENSE)

## 这是什么？

BiliGreen 是一个本地运行的 B 站视频与音频下载工具。发行版为单文件绿色软件，无需安装 Go、Python、FFmpeg 或额外播放器。双击程序后，它会自动打开只允许本机访问的网页界面。

> 请仅下载你拥有版权、获得授权或平台明确允许保存的内容。本工具不会绕过会员、付费、DRM、地区或账号权限。

如果你只是想使用，不需要阅读源码，也不需要安装开发环境：从下面的表格下载适合自己电脑的压缩包，解压后双击即可。

## 立即下载

| 系统 | 常见电脑（x64） | ARM64 设备 |
| --- | --- | --- |
| Windows | [下载 Windows x64](https://github.com/Nokia-fan/BiliGreen/releases/latest/download/BiliGreen-Windows-amd64.zip) | [下载 Windows ARM64](https://github.com/Nokia-fan/BiliGreen/releases/latest/download/BiliGreen-Windows-arm64.zip) |
| Linux | [下载 Linux x64](https://github.com/Nokia-fan/BiliGreen/releases/latest/download/BiliGreen-Linux-amd64.tar.gz) | [下载 Linux ARM64](https://github.com/Nokia-fan/BiliGreen/releases/latest/download/BiliGreen-Linux-arm64.tar.gz) |

不确定选哪个时，大多数普通 Windows 或 Linux 电脑请选择 **x64**。完整版本记录和所有文件可在 [Releases](https://github.com/Nokia-fan/BiliGreen/releases) 查看。

## 快速导航

- [主要功能](#主要功能)
- [使用方法](#使用方法)
- [下载目录](#下载目录)
- [视频和音频质量](#视频和音频质量)
- [收藏夹批量下载](#收藏夹批量下载)
- [登录与隐私](#登录与隐私)
- [常见限制](#已知限制)
- [项目创作与致谢](#项目创作与致谢)
- [参与贡献](CONTRIBUTING.md) · [安全报告](SECURITY.md) · [更新记录](CHANGELOG.md)

## 主要功能

- Windows、Linux 的 x64 与 ARM64 绿色版；源代码也可构建 macOS 版本。
- 客户端扫码登录，也支持 Cookie 备用登录；不接收账号密码。
- 即时热门、站内搜索、封面显示。
- 登录后浏览收藏夹，支持前 N 个、全部或勾选批量下载。
- 支持单P和多P视频，多P会显示总数、分P标题和各自时长。
- 根据当前视频、当前账号实时检测清晰度，支持源视频实际提供的 4K、8K、HDR、杜比视界等档位。
- DASH 音视频轨无损合并为 MP4；内置合并器，无需 FFmpeg。
- 对源投稿本身没有音轨的视频，会正常保存无声 MP4 并在任务状态中明确标注，不再误判为下载失败。
- “仅音频”下载会实时列出音轨码率和编码，默认最高音质，保存为 M4A。
- 断点续传、两个任务并发下载、任务进度与实际质量显示。
- 下载完成后可直接在网页播放；浏览器不支持某种编码时仍显示长度和分P信息。
- 收藏夹任务可在下载成功后自动保留、取消收藏或迁移至另一个收藏夹。

## 下载目录

默认目录是系统下载文件夹中的 `BiliGreen`：

- Windows：`C:\Users\你的用户名\Downloads\BiliGreen`
- Linux：`/home/你的用户名/Downloads/BiliGreen`
- macOS：`/Users/你的用户名/Downloads/BiliGreen`

实际路径会醒目显示在首页顶部。你可以直接输入绝对路径或以 `~` 开头的路径，点击“应用目录”后保存；设置只保存在当前浏览器。点击“打开下载目录”可用系统文件管理器直接打开。

视频保存为 `.mp4`，仅音频保存为 `.m4a`。文件名包含 BV 号，避免不同视频同名覆盖。未完成文件带 `.part` 后缀，再次下载同名内容时会尝试续传。

## 使用方法

1. 下载并解压与你系统和处理器匹配的压缩包。
2. Windows 双击 `BiliGreen.exe`；Linux 双击 `BiliGreen` 并在文件管理器询问时选择“运行”。
3. 浏览器会自动打开 `http://127.0.0.1:17890/`。
4. 如需会员画质或收藏夹，点击右上角“登录”，使用哔哩哔哩客户端扫码。
5. 从热门、搜索或收藏夹选择视频，在弹窗中选择分P、下载类型和质量。
6. 查看首页顶部的保存位置，点击“打开下载目录”即可找到文件。
7. 使用完毕请点击右上角“退出程序”；仅关闭网页不会停止后台程序。

重复双击程序会打开已经运行的实例。若启动失败，原因会写入程序旁的 `BiliGreen-startup-error.txt`。

## 视频和音频质量

质量列表不是固定的：程序每次都会向 B 站请求该视频在当前登录账号下真实可用的媒体流。没有登录、大会员失效、源视频未上传高画质或平台限制时，对应档位不会出现。

“最高可用”会选视频返回的最高画质；“仅音频”默认选返回音轨中码率最高的一条，也可以在单个下载弹窗中改选其他音质。M4A 保留原始 AAC/MP4 音轨，不进行有损转码。

## 收藏夹批量下载

- “加载/下载数量”默认为 20；填 `0` 会读取全部，当前安全上限为 5000 条。
- 批量类型可选择视频或仅音频；批量音频默认最高音质。
- 多P视频会把全部分P依次加入队列。
- 最多同时下载两个任务，其余显示为排队状态。
- 只有文件完整落盘后才会执行取消收藏或迁移；失败时原收藏不会被修改。
- 超大收藏夹建议每批处理 50–100 个，避免一次显示和排队数千条任务导致浏览器卡顿。

## 登录与隐私

- 扫码所得 Cookie 只保存在当前程序进程内存，不写入磁盘，退出即清除。
- Cookie 仅用于向 B 站请求登录态允许访问的内容和收藏夹操作。
- 页面服务只监听 `127.0.0.1`，局域网和公网设备无法访问。
- 账号密码登录未提供，因为它需要把明文密码交给工具，并涉及验证码和风控；扫码更安全稳定。

## 已知限制

- 不支持 DRM、会员或付费绕过、番剧专用流程、互动视频及多个传统媒体片段的拼接。
- HEVC、AV1、杜比视界和 HDR 文件能正常下载，但能否在网页内播放取决于当前浏览器和系统解码能力。
- 下载队列目前只保存在内存中；程序退出后任务列表清空，但完整文件和 `.part` 文件仍保留。
- B 站接口可能变化，若出现全部质量消失、登录失效或收藏夹错误，需要更新程序。

## 项目文件结构

```text
bili-green/
├── main.go / features.go / media.go / puremux.go / ui.go  源代码
├── build-all.sh / package-release.sh                       构建脚本
├── build/                                                  各平台原始二进制
├── release/                                                可直接分发的压缩包
└── README.md                                               本说明
```

用户下载数据默认不放在项目或发行包目录内，而是位于系统 `Downloads/BiliGreen`。

## 开发与构建

开发者需要 Go 1.22 或更高版本；最终用户不需要任何开发环境。

```sh
go run .
```

构建所有平台并生成发行包：

```sh
chmod +x build-all.sh package-release.sh
./build-all.sh
./package-release.sh
```

原始二进制位于 `build/`，Windows/Linux 发行包位于 `release/`。Windows 构建使用 GUI 子系统，不弹出常驻黑色窗口；Linux 压缩包保留可执行权限。macOS 未签名程序首次启动可能需要在系统安全设置中手动放行。

## 项目创作与致谢

BiliGreen 由 [Nokia-fan](https://github.com/Nokia-fan) 发起和维护，并在 [OpenAI Codex](https://openai.com/codex/) 的协助下完成产品设计、程序实现、问题排查、测试、文档整理与发行准备。

Codex 在本项目中是 AI 协作工具，不对应可加入 GitHub 仓库的独立个人账号。项目方向、最终决策、发布内容及维护责任由项目维护者承担。详细说明见 [CONTRIBUTORS.md](CONTRIBUTORS.md)。
