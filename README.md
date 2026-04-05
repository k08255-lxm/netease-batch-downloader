# NetEase Batch Downloader

Windows 友好的网易云歌单/专辑批量下载器。

目标很直接：让用户下载压缩包后，双击打开窗口，填最少的信息就能批量下载，并尽量保留完整元数据。

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

## 最短使用流程

1. 去 Release 下载 `netease-batch-windows-amd64.zip`
2. 解压
3. 双击 `setup_netease_batch_windows.bat`
4. 在窗口里：
   - 导入或粘贴 `MUSIC_U`
   - 粘贴歌单链接
   - 选择输出目录
   - 点击 `Check Cookie`
   - 点击 `Start Download`

不会找 `MUSIC_U` 的话，GUI 里现在有 `How To Get Cookie` 按钮，会直接弹出步骤。

## Windows GUI

首次使用和日常使用都可以直接双击：

```bat
setup_netease_batch_windows.bat
```

图形界面支持：

- 手动粘贴 `MUSIC_U`
- 从剪贴板导入 `MUSIC_U` 或整段 `Cookie:` 文本
- 导入用户自己导出的 `cookie.txt`
- 内置 `How To Get Cookie` 按钮，直接显示获取步骤
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

## 参数

- `-config`：配置文件路径，默认 `config.ini`
- `-music-u`：直接传入 `MUSIC_U`
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
- 直接粘贴浏览器里复制出来的整段 `Cookie:` 文本
- 导入用户自己导出的 `cookie.txt`

不提供对本机浏览器 Cookie 库的直接抓取。

## 封面说明

- 对 `mp3`、`flac`、`m4a`、`mp4`，程序会尝试把封面直接写进音频标签
- 同时仍会额外导出 `covers/` 下的图片文件，目的是保留源数据，便于后续导库或二次处理
- 正常播放器读取时，通常不需要再单独依赖外部图片文件
- 如果格式不支持，或者标签写入失败，音频外的封面文件仍然会保留

## 文档

- 详细说明见 `docs/netease-batch.md`

## 许可证

GPL-3.0
