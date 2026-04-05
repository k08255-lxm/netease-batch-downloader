# Upstream And License

本仓库 `netease-batch-downloader` 基于上游项目 `MusicBot-Go` 的部分 GPL-3.0 代码继续修改、裁剪和重构而来。

上游信息：

- 项目：`MusicBot-Go`
- 仓库：`https://github.com/liuran001/MusicBot-Go`
- 许可证：`GPL-3.0`

当前仓库说明：

- 当前仓库继续以 `GPL-3.0` 发布
- 上游项目的版权归原作者及贡献者所有
- 本仓库在保留相关源代码和许可证的前提下，进行了面向 `netease-batch-downloader` 的专项整理

本仓库的主要整理方向包括：

- 删除与当前产品无关的 Telegram Bot、多平台插件和历史发布链路
- 保留并重构网易云批量下载所需的下载、标签、配置和 Cookie 读取能力
- 增加 Windows 中文 GUI、浏览器 Cookie 自动读取和 GitHub Actions 自动打包发布

如果你在分发本项目，请一并保留：

- `LICENSE`
- 本仓库完整对应源码
- 本文件 `UPSTREAM.md`
