# 网易云歌单批量下载

这是一个独立的 `netease-batch-downloader` 仓库，只面向网易云歌单/专辑批量下载。

核心命令：

```bash
go build -o netease-batch ./cmd/netease-batch
```

它专门用来批量下载网易云歌单/专辑，不再承担 Telegram Bot、Docker 镜像或多平台音乐聚合发布职责。

## 能力

- 输入网易云歌单或专辑 URL，批量下载全部歌曲
- 支持 `standard` / `high` / `lossless` / `hires`
- 自动写入音频标签：
  - 标题
  - 艺术家
  - 专辑
  - 专辑艺术家
  - 年份
  - 曲号 / 碟号
  - 封面
  - 歌词
- 额外导出 sidecar 文件：
  - `playlist.json`
  - 每首歌对应的 `*.json`
  - 每首歌对应的 `*.lrc`
  - `covers/` 里的封面文件

## Windows 快速使用

1. 安装 Go 1.26+
2. 在仓库根目录执行配置向导：

```bat
setup_netease_batch_windows.bat
```

它会自动：

- 打开一个图形界面窗口，用户直接填表，不需要命令行输入
- 构建 `netease-batch.exe`
- 创建 `config.ini`
- 打开网易云登录页
- 支持粘贴 `MUSIC_U`
- 在未手动配置 Cookie 时自动尝试从浏览器读取 `music.163.com` 登录态
- 支持粘贴整段 `Cookie:` 文本并自动提取 `MUSIC_U`
- 支持导入用户自己导出的 `cookie.txt`
- 内置“如何获取 Cookie”按钮，直接弹窗教用户怎么拿
- 自动写回 `[plugins.netease] music_u`
- 立即校验 Cookie 是否可用
- 可选直接开始第一次歌单下载

3. 之后日常下载：

```bat
download_netease_playlist.bat "https://music.163.com/#/playlist?id=19723756" "D:\Music"
```

如果你想继续用图形界面，也可以每次都直接双击：

```bat
setup_netease_batch_windows.bat
```

界面里直接点：

- `打开网易云`
- `如何获取 Cookie`
- `导入剪贴板`
- `导入 Cookie 文件`
- `检查 Cookie`
- `开始下载`

默认会输出到：

```text
<输出目录>\<歌单名>\
  playlist.json
  playlist-cover.jpg
  tracks\
  covers\
```

## 直接运行

```bash
go run ./cmd/netease-batch \
  -config config.ini \
  -url "https://music.163.com/#/playlist?id=19723756" \
  -out downloads \
  -quality lossless \
  -concurrency 4
```

## 常用参数

- `-config`：配置文件路径，默认 `config.ini`
- `-music-u`：直接传入 `MUSIC_U`，优先级高于配置文件
- 如果配置里没有 `cookie` / `music_u`，Windows 下会自动尝试读取 `Edge` / `Chrome` / `Brave` / `Firefox`
- `-url`：歌单或专辑 URL
- `-out`：输出根目录
- `-quality`：`standard` / `high` / `lossless` / `hires`
- `-concurrency`：同时下载歌曲数
- `-lyrics=false`：不导出歌词文件
- `-covers=false`：不导出封面文件
- `-overwrite=true`：覆盖已存在文件
- `-check`：只校验 `MUSIC_U` 是否可用，不下载

## Release 自动化

这个仓库的 GitHub Actions 会直接构建并发布 `netease-batch`：

- `netease-batch-windows-amd64-vX.Y.Z.zip`
- `netease-batch-linux-amd64-vX.Y.Z.tar.gz`
- `netease-batch-linux-arm64-vX.Y.Z.tar.gz`
- `netease-batch-darwin-amd64-vX.Y.Z.tar.gz`
- `netease-batch-darwin-arm64-vX.Y.Z.tar.gz`

打 tag 后会自动创建或更新 GitHub Release，不再发布历史遗留的 `musicbot-go` 产物。

## 说明

- `playlist.json` 记录整个歌单下载结果
- 每首歌旁边的 `*.json` 额外保留源数据，便于后续导库或二次处理
- 如果文件已存在，默认跳过，不会重复下载
- Windows 首次使用建议直接运行 `setup_netease_batch_windows.bat`
- 现在的 Windows 向导是 GUI 方式，适合“下载后双击即填表使用”
- 如果没有手动配置 `cookie` / `music_u`，Windows 下程序会自动尝试从已登录浏览器读取 `music.163.com` Cookie
- 对 `mp3`、`flac`、`m4a`、`mp4`，封面会尝试直接写进音频标签；外部 `covers/` 图片仍会保留作为源数据备份
