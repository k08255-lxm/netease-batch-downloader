# 网易云歌单批量下载

这个仓库新增了一个独立命令：

```bash
go build -o netease-batch ./cmd/netease-batch
```

它专门用来批量下载网易云歌单/专辑，不走 Telegram Bot 的交互分页流程。

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

- 构建 `netease-batch.exe`
- 创建 `config.ini`
- 打开网易云登录页
- 引导你去复制 `MUSIC_U`
- 自动写回 `[plugins.netease] music_u`
- 立即校验 Cookie 是否可用
- 可选直接开始第一次歌单下载

3. 之后日常下载：

```bat
download_netease_playlist.bat "https://music.163.com/#/playlist?id=19723756" "D:\Music"
```

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
- `-url`：歌单或专辑 URL
- `-out`：输出根目录
- `-quality`：`standard` / `high` / `lossless` / `hires`
- `-concurrency`：同时下载歌曲数
- `-lyrics=false`：不导出歌词文件
- `-covers=false`：不导出封面文件
- `-overwrite=true`：覆盖已存在文件
- `-check`：只校验 `MUSIC_U` 是否可用，不下载

## 说明

- `playlist.json` 记录整个歌单下载结果
- 每首歌旁边的 `*.json` 额外保留源数据，便于后续导库或二次处理
- 如果文件已存在，默认跳过，不会重复下载
- Windows 首次使用建议直接运行 `setup_netease_batch_windows.bat`
