# weekly-planner

Personal Telegram bot for weekly task planning.

## Configuration

Create a `.env` file (or set environment variables):

```
BOT_TOKEN=<telegram bot token>
OWNER_TELEGRAM_ID=<your telegram user id>
DATABASE_PATH=bot.db       # optional, default: bot.db
HTTP_PORT=8080             # optional, default: 8080
```

## Run locally

```sh
go run ./cmd/bot
```

Sandbox mode (no token required):

```sh
make run
```

## Deploy

```sh
make deploy SERVER=opc@143.47.97.133
```

This builds for Linux, uploads to `/tmp/bot-new`, moves it to `/usr/local/bin/weekly-planner-bot`, fixes the SELinux context, and restarts the service.

## Server layout

| Path | What |
|------|------|
| `/usr/local/bin/weekly-planner-bot` | binary |
| `/etc/weekly-planner.env` | env config (BOT_TOKEN etc.) |
| `/etc/systemd/system/weekly-planner.service` | systemd unit |
| `/var/lib/weekly-planner/bot.db` | database |

## Server setup (first time)

```sh
scp .env opc@143.47.97.133:/etc/weekly-planner.env
```

Systemd unit (`/etc/systemd/system/weekly-planner.service`):

```ini
[Unit]
Description=Weekly Planner Bot
After=network.target

[Service]
User=opc
EnvironmentFile=/etc/weekly-planner.env
ExecStart=/usr/local/bin/weekly-planner-bot
WorkingDirectory=/var/lib/weekly-planner
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```sh
sudo mkdir -p /var/lib/weekly-planner
sudo systemctl daemon-reload
sudo systemctl enable --now weekly-planner
```

## DB backup before touching anything

```sh
ssh opc@143.47.97.133 "sudo cp /var/lib/weekly-planner/bot.db /var/lib/weekly-planner/bot.db.bak"
```
