# Flamego Stats

看看哪些 GitHub 项目在使用 [Flamego](https://github.com/flamego/flamego)。

定时收集 GitHub 公开仓库的数据，展示 Flamego 的使用情况。后端基于 Go、Flamego 和 PostgreSQL，前端使用 Vue 3 和 TypeScript。

[English](README.md) · [Flamego 文档](https://flamego.dev/) · [API](docs/swagger.yaml)

## ✨ 功能

- 依赖仓库列表、Star 排行和使用者排行。
- Flamego 版本、Go 版本、中间件、语言和 Topic 分布。
- 仓库搜索、筛选和排序，可选择是否包含 fork。
- 依赖仓库数变化趋势。
- 中英文界面，支持浅色和深色主题。

## 🚀 部署

需要 Docker 和 Docker Compose。Compose 会启动 Web 服务、同步 worker 和 PostgreSQL。

```sh
git clone https://github.com/wuhan005/flamego-stats.git
cd flamego-stats
cp config/flamego-stats.example.yaml config/flamego-stats.yaml
```

编辑 `config/flamego-stats.yaml`：将 `postgres.dsn` 中的主机改为 `postgres`，数据库密码与 `docker-compose.yaml` 中的 `POSTGRES_PASSWORD` 保持一致。部署前请替换默认密码 `change-me`，并填入 `github.token`，减少同步时的 API 限流。

```sh
docker compose up -d
```

打开 [http://localhost:8080](http://localhost:8080)。worker 首次启动后会同步数据，之后默认每 6 小时更新一次。完整配置见 [示例文件](config/flamego-stats.example.yaml)。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request。报告问题时请附上复现步骤，每个 PR 尽量只处理一件事。
