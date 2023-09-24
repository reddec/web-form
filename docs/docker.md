# Docker

By default, in docker mode:

- database migrations are enabled and expected in `/migrations` dir
- storage type is `files` and storage path is `/data`
- configurations should be mounted to `/configs`

If storage is `database` and migration directory (`DB_MIGRATIONS`, default is `/migrations`) contains at least one
SQL file (`.sql`) then migration will be applied automatically
using [sql-migrate](https://github.com/rubenv/sql-migrate).


For example

    docker run -v $(pwd):/migrations:ro -e STORAGE=database -e DB_URL=:memory: -e DB_DIALECT=sqlite3 ghcr.io/reddec/web-form:latest

It also simplifies Kubernetes deployment, since you can use config maps for migrations and mount them as a volume.