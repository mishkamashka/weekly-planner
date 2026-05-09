# weekly-planner

Personal Telegram bot for weekly task planning.

## Configuration

Environment variables (set in `/etc/weekly-planner/env` on server, or `.env` locally):

```
BOT_TOKEN=<telegram bot token>
OWNER_TELEGRAM_ID=<your telegram user id>
DATABASE_PATH=bot.db       # optional, default: bot.db
```

## Make commands

| Command | Description |
|---------|-------------|
| `make run` | Run locally in sandbox mode (no token or DB needed) |
| `make build` | Compile binary for current OS → `bin/bot` |
| `make build-linux` | Cross-compile for Linux amd64 → `bin/bot-linux` |
| `make deploy SERVER=opc@<ip>` | Build, upload binary + service file, reload and restart |
| `make tidy` | Tidy Go modules |
| `make clean` | Remove `bin/` |

Deploy example:
```sh
make deploy SERVER=opc@143.47.97.133
```

## Server layout (Oracle Cloud AMD Micro, OL9, Ashburn)

SSH: `ssh opc@143.47.97.133`

| Path | What |
|------|------|
| `/usr/local/bin/weekly-planner-bot` | binary |
| `/etc/systemd/system/weekly-planner.service` | systemd unit |
| `/etc/weekly-planner/env` | env config (BOT_TOKEN etc.) |
| `/var/lib/weekly-planner/bot.db` | SQLite database |
| `/var/lib/weekly-planner/bot.db-shm` | SQLite shared memory (WAL mode) |
| `/var/lib/weekly-planner/bot.db-wal` | SQLite WAL file |

## Service management

```sh
sudo systemctl status weekly-planner
sudo systemctl restart weekly-planner
sudo journalctl -u weekly-planner -f
```

## DB backup before touching anything

```sh
ssh opc@143.47.97.133 "cp /var/lib/weekly-planner/bot.db /var/lib/weekly-planner/bot.db.bak"
```
