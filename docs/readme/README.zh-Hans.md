<p align="center">
  <a href="../../README.md">English</a> ·
  <a href="README.zh-Hans.md">简体中文</a>
</p>

# Pikapu

安静、简洁的自托管个人 RSS 阅读器。Pikapu 汇集你关心的 RSS、Atom 与 JSON Feed，以清爽的三栏布局呈现。它以单个 Docker 容器运行：Go 后端内嵌 React 前端与 SQLite 数据库，只占用一个端口。

> [!IMPORTANT]
> Pikapu 仍处于早期阶段。升级前请备份 `data/`；在本机之外开放访问前，请设置 `PIKAPU_PASSWORD`。

## 主要功能

- 输入订阅地址或网站首页即可订阅，自动发现订阅源
- 分类、未读计数、星标、搜索，以及“未读 / 全部”筛选
- 关键词过滤规则：按关键词将新文章标为已读或直接跳过，可作用于全部订阅源或单个订阅源
- 两种布局：经典三栏阅读器，或“消息中心”——先看为你推荐的文章，再看按日期与来源归组的更新；推荐会参考你常读和加星标的订阅源
- 桌面三栏、手机单栏，浅色 / 深色主题，正文字号调节，键盘快捷键；可添加到手机主屏幕或安装到桌面，像应用一样使用
- 服务端安全过滤正文 HTML，修复相对链接与懒加载图片，缓存站点图标
- 后台定时刷新（ETag / Last-Modified 条件请求，失败自动退避），自动清理过期的已读文章，星标文章始终保留
- OPML 导入 / 导出
- 支持英语与简体中文：默认跟随浏览器语言，不支持的语言回退到英语，也可在设置中手动切换
- 可选的访问密码

## 快速开始

需要 Docker 与 Compose。

```sh
git clone <repository-url> pikapu
cd pikapu
docker compose up -d --build
```

打开 <http://localhost:7660>，点击 **+** 添加第一个订阅。

如需密码保护，将 [`.env.example`](../../.env.example) 复制为 `.env`，设置 `PIKAPU_PASSWORD` 后再次执行 `docker compose up -d`。全部选项见[配置文档](../operations/configuration.md)（英文）。

默认端口映射监听所有网卡。若不希望局域网访问，请绑定到 `127.0.0.1`、使用带 TLS 的反向代理或可信 VPN，详见[安全文档](../operations/security.md)。

## 运行时数据

| 主机目录 | 容器目录 | 用途 | 需要备份 |
| --- | --- | --- | --- |
| `./data` | `/data` | SQLite 数据库（`pikapu.db` 及 `-wal` / `-shm` 文件） | 是 |

请勿提交 `data/` 或 `.env`，其中包含订阅、阅读记录和会话密钥。

## 文档

完整文档（英文）见[文档索引](../README.md)。开发前请阅读 [CONTRIBUTING.md](../../CONTRIBUTING.md) 与 [AGENTS.md](../../AGENTS.md)。测试默认在 Docker 中运行：

```sh
make help   # 查看所有目标
make test   # 在容器中运行前后端检查，并对镜像做冒烟测试
```

## 安全

漏洞请按 [SECURITY.md](../../SECURITY.md) 私下报告。
