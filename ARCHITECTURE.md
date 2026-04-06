# 架构说明

## 仓库定位

本仓库现在是一个专门的 `netease-batch-downloader` 仓库。

对外可运行的程序只有一个：

- `cmd/netease-batch`

它负责：

- 读取网易云歌单或专辑
- 批量下载歌曲
- 写入元数据、封面、歌词
- 导出 `playlist.json` 和每首歌的 sidecar 文件

仓库中保留的 `bot/*` 目录只是下载器内部复用库，名称来自历史代码演进，不再代表一个对外 Bot 产品。

## 目录结构

```text
netease-batch-downloader/
├── cmd/netease-batch/              # 批量下载器主程序
├── plugins/netease/                # 网易云 API / Cookie / URL 解析
├── bot/download/                   # 下载器（重试、超时、多线程）
├── bot/id3/                        # 音频标签写入
├── bot/platform/                   # 通用类型与音质抽象
├── bot/httpproxy/                  # 上游 API 代理支持
├── docs/netease-batch.md           # 使用文档
├── config_example.ini              # 下载器配置模板
├── build.sh                        # 命令行构建脚本
├── build_netease_batch_windows.bat # Windows 构建脚本
├── setup_netease_batch_windows.*   # Windows 图形化入口
└── .github/workflows/ci.yml        # 仅为下载器服务的 CI / Release
```

## 核心流程

### 1. 启动

```text
cmd/netease-batch/main.go
  ├─> 解析命令行参数
  ├─> 读取 config.ini
  ├─> 解析 [plugins.netease] 配置
  ├─> 如果未手填 Cookie，则在 Windows 下尝试自动读取浏览器 Cookie
  ├─> 初始化网易云客户端
  └─> 执行 Cookie 校验或批量下载
```

### 2. 批量下载

```text
歌单/专辑 URL
  └─> plugins/netease.MatchPlaylistURL()
       └─> plugins/netease.GetPlaylist()
            └─> 遍历曲目
                 ├─> GetTrack / GetDownloadInfo
                 ├─> bot/download.DownloadService
                 ├─> plugins/netease.ID3Provider
                 ├─> bot/id3.ID3Service
                 └─> 生成封面、歌词、sidecar 与 playlist.json
```

### 3. Cookie 来源优先级

```text
1. -music-u
2. [plugins.netease] cookie
3. [plugins.netease] music_u
4. Windows 浏览器自动读取（Edge / Chrome / Brave / Firefox）
```

## 发布策略

GitHub Actions 只构建并发布 `netease-batch` 相关产物：

- `netease-batch-windows-amd64-latest.zip`
- `netease-batch-windows-amd64-vX.Y.Z.zip`

推送到 `main` 后，GitHub 会自动测试、打包并更新滚动 `latest` Release。

推送 `v*` tag 后，GitHub 会自动测试、打包并更新对应正式版 Release。
