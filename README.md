# Flamego Stats

See which GitHub projects use [Flamego](https://github.com/flamego/flamego).

Collects data from public GitHub repositories on a schedule and shows how Flamego is used. Built with Go, Flamego, and PostgreSQL, with a Vue 3 and TypeScript frontend.

[简体中文](README.zh-CN.md) · [Flamego Docs](https://flamego.dev/) · [API](docs/swagger.yaml)

## ✨ Features

- Dependent repositories, star rankings, and top users and organizations.
- Flamego versions, Go versions, middleware, languages, and topics.
- Repository search, filters, and sorting, with an option to include forks.
- Trends in the number of dependent repositories.
- English and Chinese interface with light and dark themes.

## 🚀 Deployment

You need Docker and Docker Compose. Compose starts the web server, sync worker, and PostgreSQL.

```sh
git clone https://github.com/wuhan005/flamego-stats.git
cd flamego-stats
cp config/flamego-stats.example.yaml config/flamego-stats.yaml
```

Edit `config/flamego-stats.yaml`: set the host in `postgres.dsn` to `postgres` and use the same password as `POSTGRES_PASSWORD` in `docker-compose.yaml`. Replace the default password `change-me` before deployment and set `github.token` to reduce API rate limits during synchronization.

```sh
docker compose up -d
```

Open [http://localhost:8080](http://localhost:8080). The worker syncs data on first startup, then every 6 hours by default. See the [example configuration](config/flamego-stats.example.yaml) for all options.

## 🤝 Contributing

Issues and pull requests are welcome. Include steps to reproduce bugs and keep each PR focused on one change.
