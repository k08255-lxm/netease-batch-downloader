# NetEase Batch Downloader

Windows 友好的网易云歌单/专辑批量下载器。

目标很直接：让用户下载压缩包后，双击打开窗口，填最少的信息就能批量下载，并尽量保留完整元数据。

这个仓库现在只保留 `netease-batch-downloader` 本身需要的代码、脚本和自动化流程，不再混放历史遗留的其它产品代码或发布链路。

仓库地址：

- GitHub: `https://github.com/k08255-lxm/netease-batch-downloader`

Release 下载：

- `https://github.com/k08255-lxm/netease-batch-downloader/releases/latest`

## 功能

- 批量下载网易云歌单
- 批量下载网易云专辑
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
- 导出额外源数据：
  - `playlist.json`
  - 每首歌旁边的 `*.json`
  - 每首歌旁边的 `*.lrc`
  - `covers/` 里的封面文件

## 仓库结构

- `cmd/netease-batch`：命令行主程序
- `plugins/netease`：网易云接口、Cookie、URL 解析、识曲适配
- `bot/download`：下载执行与并发控制
- `bot/id3`：封面、歌词、元数据写入
- `bot/platform`：下载器内部通用类型
- `.github/workflows/ci.yml`：测试、打包、发布 `netease-batch`

## 最短使用流程

1. 去 Release 下载最新的 `netease-batch-windows-amd64-vX.Y.Z.zip`
2. 解压
3. 双击 `setup_netease_batch_windows.bat`
4. 在窗口里：
   - 直接粘贴 `MUSIC_U`，或留空让程序自动从浏览器读取
   - 粘贴歌单链接
   - 选择输出目录
   - 点击“检查 Cookie”
   - 点击“开始下载”

Windows 下如果配置里没有手动填写 `cookie` / `music_u`，程序会自动尝试从已登录的 `Edge`、`Chrome`、`Brave`、`Firefox` 读取 `music.163.com` 的 Cookie。

## Windows GUI

首次使用和日常使用都可以直接双击：

```bat
setup_netease_batch_windows.bat
```

图形界面支持：

- 手动粘贴 `MUSIC_U`
- `MUSIC_U` 留空时运行时自动尝试读取浏览器 Cookie
- 从剪贴板导入 `MUSIC_U` 或整段 `Cookie:` 文本
- 导入用户自己导出的 `cookie.txt`
- 内置“如何获取 Cookie”按钮，直接显示获取步骤
- 一键校验 Cookie
- 一键开始下载
- 打开输出目录
- 打开 `config.ini`

## 命令行

如果你想自己跑命令，也可以直接构建：

```bash
go build -o netease-batch ./cmd/netease-batch
```

示例：

```bash
go run ./cmd/netease-batch \
  -config config.ini \
  -url "https://music.163.com/#/playlist?id=19723756" \
  -out downloads \
  -quality lossless \
  -concurrency 4
```

## Release 产物

GitHub Release 现在只发布 `netease-batch` 相关产物。

当前发布命名：

- Windows: `netease-batch-windows-amd64-vX.Y.Z.zip`
- Linux: `netease-batch-linux-amd64-vX.Y.Z.tar.gz`
- Linux ARM64: `netease-batch-linux-arm64-vX.Y.Z.tar.gz`
- macOS: `netease-batch-darwin-amd64-vX.Y.Z.tar.gz`
- macOS ARM64: `netease-batch-darwin-arm64-vX.Y.Z.tar.gz`

## 参数

- `-config`：配置文件路径，默认 `config.ini`
- `-music-u`：直接传入 `MUSIC_U`
- 未显式提供 `cookie`/`music_u` 时，Windows 下会自动尝试从浏览器读取
- `-url`：歌单或专辑 URL
- `-out`：输出根目录
- `-quality`：`standard` / `high` / `lossless` / `hires`
- `-concurrency`：同时下载歌曲数
- `-lyrics=false`：不导出歌词
- `-covers=false`：不导出封面
- `-overwrite=true`：覆盖已存在文件
- `-check`：只校验 Cookie，不下载

## 输出结构

```text
<输出目录>\<歌单名>\
  playlist.json
  playlist-cover.jpg
  tracks\
    001 - 艺术家 - 歌名.flac
    001 - 艺术家 - 歌名.json
    001 - 艺术家 - 歌名.lrc
  covers\
    123456789.jpg
```

## Cookie 说明

这个工具需要用户自己的网易云登录态 `MUSIC_U` 才能下载高质量资源。

为了尽量减少操作，GUI 现在支持：

- 直接粘贴 `MUSIC_U`
- Windows 下在未手动配置 Cookie 时自动读取浏览器中的 `music.163.com` 登录态
- 直接粘贴浏览器里复制出来的整段 `Cookie:` 文本
- 导入用户自己导出的 `cookie.txt`
- 也可以在 `[plugins.netease]` 中直接配置整段 `cookie`

## 封面说明

- 对 `mp3`、`flac`、`m4a`、`mp4`，程序会尝试把封面直接写进音频标签
- 同时仍会额外导出 `covers/` 下的图片文件，目的是保留源数据，便于后续导库或二次处理
- 正常播放器读取时，通常不需要再单独依赖外部图片文件
- 如果格式不支持，或者标签写入失败，音频外的封面文件仍然会保留

## 文档

- 详细说明见 `docs/netease-batch.md`
- 上游来源与许可证说明见 `UPSTREAM.md`

## 上游来源

这个仓库基于上游项目 `liuran001/MusicBot-Go` 的 GPL-3.0 代码继续修改而来，并已重构为专门的 `netease-batch-downloader` 仓库。

- 上游仓库：`https://github.com/liuran001/MusicBot-Go`
- 上游许可证：`GPL-3.0`
- 当前仓库继续以 `GPL-3.0` 分发

上游项目的版权归原作者及其贡献者所有；本仓库对保留下来的相关代码做了裁剪、重命名、中文化、自动化发布和产品定位重构。

## 许可证

GPL-3.0
