# Weekly Planner Telegram Bot — Build Plan

A personal weekly-planner Telegram bot, built in Go, that helps you distribute backlog tasks across the week, reminds you each morning, and lets you check tasks off with one tap.

## Product summary

**Core loop:**
1. You maintain a **backlog** of tasks (one big pool).
2. Every **Sunday evening**, the bot pings you and walks you through assigning backlog items to specific days of the upcoming week.
3. **Recurring presets** (e.g., "Training on Tuesday") are auto-placed before you start assigning.
4. Each **weekday morning**, the bot sends you that day's list with a "done" button next to each item.
5. (v2) On **Sunday**, a retrospective: "You planned 10, finished 7. Try 7 next week."

**Principles for v1:**
- Single user (you), but data model is multi-user-safe from day 1.
- Run locally or on a cheap server — no premature ops work.
- Dogfood hard for 2–3 weeks before inviting friends. The bot will change a lot based on your real usage.

---

## Tech stack

| Concern | Choice | Why |
|---|---|---|
| Language | Go 1.22+ | Your pick. Single-binary deploys are ideal for Telegram bots. |
| Telegram lib | `github.com/go-telegram/bot` | Clean API, active, good update/callback routing. |
| Database | SQLite via `modernc.org/sqlite` | Pure Go (no CGo). Plenty fast for hundreds of users. Single file backup. |
| Migrations | `github.com/pressly/goose/v3` | Simple, embeddable. |
| Scheduling | `github.com/robfig/cron/v3` | Cron expressions for daily/weekly jobs. |
| Config | `github.com/joho/godotenv` + env vars | Keep token out of code. |
| Logging | `log/slog` (stdlib) | No extra deps. |

**Library decision note:** `go-telegram/bot` uses long-polling by default, which is what you want for v1. Webhooks are a premature optimization until you're on a proper server with TLS.

---

## Hosting recommendation (free)

The bot needs to be **always-on** so scheduled notifications fire. That rules out anything that sleeps on idle (Render free, Railway free, Cloud Run without min-instance).

**Recommendation: Oracle Cloud Infrastructure "Always Free" — AMD Micro, Frankfurt (`eu-frankfurt-1`).**

Verified against Oracle's current Always Free docs:
- **VM.Standard.E2.1.Micro** shape: 1/8 OCPU + 1 GB RAM, 50 GB boot volume (from a 200 GB total block storage pool)
- **10 TB/month** outbound data transfer (huge compared to GCP's 1 GB)
- Low European latency (Frankfurt is 1 hop from most of Central/Western Europe)
- No end date on Always Free
- Bonus you won't use but comes free: 20 GB object storage, 1 flexible load balancer, 2 Autonomous Databases

Sizing check for this bot: Go binary ~20–50 MB resident, SQLite negligible, scheduled jobs idle most of the time. 1 GB RAM is tight but sufficient.

### Why Oracle over Google Cloud for this use case

The deciding factor: when Oracle's 30-day $300 trial ends, your account **automatically** reverts to Always Free — no manual upgrade needed. On GCP, you must click "Upgrade to Paid billing account" before the trial ends or everything gets shut down, even if your usage is inside free tier limits. For a low-touch personal project, Oracle's auto-transition is much less error-prone.

### Oracle signup / operation gotchas

1. **Idle instance reclamation is real.** Oracle can reclaim Always Free compute if, over a 7-day window, CPU 95p < 20% AND network < 20% (AND memory < 20% for A1 shapes only). A personal bot is idle 99% of the time, so this matters.
    - **Mitigation A (shape choice):** Use the **AMD Micro** — it has no memory utilization check, so hitting one of the three thresholds is easier.
    - **Mitigation B (keepalive):** run a lightweight internal cron every ~5 minutes (touch a file, do a trivial computation, ping `api.telegram.org`) to keep CPU/network averages above 20%. Cheap insurance.
    - Do both for the first few months.
2. **Home region is permanent.** You pick it once during signup and can never change it. **Pick `eu-frankfurt-1` (Frankfurt) or `eu-amsterdam-1` (Amsterdam).**
3. **A1 (ARM) capacity is spotty.** "Out of host capacity" errors are routine for A1 in Frankfurt. AMD Micro is reliably available. If you later want A1 for headroom, expect to retry provisioning.
4. **Console is dense.** You'll touch it mostly during initial setup. Bookmark the Compute Instances page and ignore the rest.
5. **Outbound port 25 is blocked.** Doesn't affect Telegram; mentioned only because people sometimes try to add `msmtp` for alerts — use Telegram DM or webhooks for alerts instead.

### Alternatives

- **Google Cloud Compute Engine `e2-micro` Always Free** — technically viable but has the 90-day-trial-must-manually-upgrade gotcha; and 1 GB/month egress (vs Oracle's 10 TB) leaves zero slack if you later add anything else to the box.
- **Your own always-on hardware** — Raspberry Pi, old laptop, NAS. Long-polling bots make outbound connections only, so no port forwarding, no TLS, no public IP. Zero infra cost. Caveat: home internet/power reliability.

### Not free (avoid for this)

- Render, Railway — free tiers sleep on idle, breaks scheduled jobs.
- Fly.io — removed the real free tier in late 2024; now ~$5/mo minimum.
- AWS Free Tier — t2.micro free for 12 months then billed.

### Upgrade path later

If friends-phase turns into something real and Oracle's reclamation rules get annoying, a **Hetzner CX22 (~€4/mo, Falkenstein/Helsinki)** is the natural next step. Same deployment shape — systemd + SQLite file — just with a better SLA and no reclamation games.

### Deployment shape (Oracle AMD Micro)

- Ubuntu 22.04 LTS (Always Free-eligible image — no licensing fees)
- systemd unit running the Go binary as a non-root user
- SQLite file in `/var/lib/weekly-planner/bot.db`
- Nightly `sqlite3 bot.db ".backup bot.db.bak"` + rotate
- Optional: upload weekly backup to the free 20 GB Object Storage bucket
- Internal 5-minute keepalive job inside the bot process to dodge idle reclamation
- Security list / firewall: only allow outbound 443 (Telegram API) and 22 from your home IP for SSH. No inbound needed.

---

## Data model

Build multi-user from day 1. Schema sketch:

```sql
users (
  id             INTEGER PRIMARY KEY,
  telegram_id    INTEGER UNIQUE NOT NULL,
  name           TEXT,
  timezone       TEXT NOT NULL DEFAULT 'Europe/Berlin',
  morning_time   TEXT NOT NULL DEFAULT '08:00',  -- HH:MM local
  sunday_time    TEXT NOT NULL DEFAULT '18:00',
  created_at     TIMESTAMP
);

tasks (
  id         INTEGER PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id),
  title      TEXT NOT NULL,
  notes      TEXT,
  status     TEXT NOT NULL,  -- 'backlog' | 'assigned' | 'done' | 'archived'
  created_at TIMESTAMP,
  done_at    TIMESTAMP
);

assignments (
  id          INTEGER PRIMARY KEY,
  task_id     INTEGER NOT NULL REFERENCES tasks(id),
  user_id     INTEGER NOT NULL,
  week_start  DATE NOT NULL,     -- Monday of the week
  day_of_week INTEGER NOT NULL,  -- 0=Mon..6=Sun
  completed   BOOLEAN DEFAULT 0,
  completed_at TIMESTAMP
);

presets (
  id          INTEGER PRIMARY KEY,
  user_id     INTEGER NOT NULL,
  title       TEXT NOT NULL,
  day_of_week INTEGER NOT NULL,  -- which day it auto-places on
  active      BOOLEAN DEFAULT 1
);
```

**Why `assignments` is separate from `tasks`:** lets a one-off task belong to one day; lets presets expand into a fresh row every week; makes "what did I do the week of April 20?" a simple query.

---

## Bot commands (v1)

| Command | Does |
|---|---|
| `/start` | Onboarding: name, timezone, morning reminder time. |
| `/add <text>` | Add task to backlog. Also works as a plain text message. |
| `/backlog` | Show backlog with inline buttons per item: assign to day / archive. |
| `/today` | Show today's assigned tasks with ✅ buttons. |
| `/week` | Weekly board: Mon–Sun, each day with its tasks. |
| `/plan` | Manually trigger the Sunday planning flow. |
| `/preset` | Manage recurring presets. |
| `/settings` | Change timezone and reminder times. |
| `/help` | Command list. |

**Inline button design:** every task card carries a `cb:task:<id>:<action>` callback. Keep callback data under Telegram's 64-byte limit — use short keys.

---

## Scheduler jobs

Run inside the bot process. Each user gets their own cron entries because reminder times are per-user.

| Job | When | What |
|---|---|---|
| Weekly plan ping | Sunday at `users.sunday_time` | DM user: "Ready to plan next week? /plan". Auto-expand presets into next week's assignments. |
| Morning reminder | Every day at `users.morning_time` | DM user today's assignments as a card with ✅ buttons. |
| (v2) Sunday recap | Sunday at `sunday_time - 30min` | Stats for the week just ending. |

**Implementation note:** on bot start, load all users and register cron entries. On `/settings` change, reload that user's entries.

---

## Project layout

```
cmd/bot/main.go                # wire + start
internal/config/config.go      # env parsing
internal/store/
  store.go                     # sqlite setup, goose migrations
  users.go tasks.go assignments.go presets.go
internal/bot/
  bot.go                       # handler registration
  handlers_commands.go         # /add, /backlog, /today...
  handlers_callbacks.go        # button callbacks
  flows_planning.go            # multi-step Sunday planning flow
  render.go                    # message/keyboard builders
internal/scheduler/scheduler.go
migrations/                    # goose .sql files
```

---

## Phased plan

Each phase should be shippable and dogfoodable on its own. Don't move to the next until the current one feels good for at least a few days.

### Phase 0 — Setup (½ day)
- [x] Create bot with [@BotFather](https://t.me/BotFather), grab token.
- [x] `go mod init`, pull dependencies.
- [x] `.env` with `BOT_TOKEN`, `DATABASE_PATH`, `OWNER_TELEGRAM_ID`.
- [x] Project skeleton, logging, graceful shutdown on SIGINT.
- [x] Implement `/start` and `/ping`. Verify round-trip works.

**Done when:** you DM the bot `/ping` and get `pong`.

### Phase 1 — Backlog (½–1 day)
- [x] SQLite + goose wired up. First migration creates `users` and `tasks`.
- [x] On `/start` or first message: auto-create a `users` row.
- [x] Persistent reply keyboard with "➕ Add task" button.
- [x] Tapping the button puts bot in "waiting for task" state; user replies with text → added to backlog.
- [x] `/add <text>` shortcut also works.
- [ ] `/backlog` renders list with per-row inline keyboard: `Mon Tue Wed Thu Fri Sat Sun Archive`.
- [ ] Day button → creates `assignments` row for the current week, flips task status to `assigned`.

**Done when:** you can add tasks and shove them around the week via buttons.

### Phase 2 — Daily view + done buttons (½ day)
- [ ] `/today` query: today's assignments for this user.
- [ ] Each task rendered as a line with a ✅ button.
- [ ] Tapping ✅ sets `assignments.completed = 1`, edits the message in place so the line shows ~~strikethrough~~.
- [ ] `/week` shows the Mon–Sun board.

**Done when:** you open `/today`, tick things off as you do them, and it feels satisfying.

### Phase 3 — Scheduled reminders (½–1 day)
- [ ] Add `timezone` and `morning_time` columns (migration).
- [ ] `/settings` command to set them.
- [ ] Scheduler: each morning at local `morning_time`, DM `/today`'s content.
- [ ] Scheduler: Sunday at local `sunday_time`, DM "Time to plan next week — tap /plan".

**Done when:** the bot pings you unprompted at 8am with your list.

### Phase 4 — Weekly planning flow (1 day)
- [ ] `/plan` starts a conversational flow: shows each backlog item, inline buttons `Mon Tue Wed Thu Fri Sat Sun Skip Archive`.
- [ ] Tracks flow state per-user in memory (map keyed by telegram_id).
- [ ] Final screen: summary of the week with ability to swap days.

**Done when:** Sunday evening feels like a 5-minute ritual instead of a chore.

### Phase 5 — Recurring presets (½ day)
- [ ] `presets` table.
- [ ] `/preset` menu: list, add ("Training" + day), remove, toggle active.
- [ ] On new week rollover (or at start of `/plan`), auto-insert assignment rows from active presets.

**Done when:** Tuesday training shows up every week without you thinking about it.

### Phase 6 — Dogfood (2–3 weeks, no code)
Use it. Write down every friction point. Don't fix anything for the first week — just note it. Likely candidates:
- "I want to move a task to next week."
- "I want to add a task straight to tomorrow, skipping backlog."
- "I want the morning reminder earlier on weekends."
- "Carry-over of unfinished tasks should be automatic."

Then fix the top 3.

### Phase 7 — Friends beta
- [ ] Deploy to Hetzner CX22 (systemd unit, env file, SQLite on disk, nightly backup).
- [ ] Light onboarding in `/start` (timezone detection, quick how-to).
- [ ] Per-user error reporting — bot shouldn't crash the process; log + reply "something went wrong".
- [ ] Invite 2–3 friends. Watch where they get stuck.

---

## v2 ideas (deferred)

- **Voice task input.** When in "waiting for task" state, accept voice messages → download audio from Telegram → transcribe via OpenAI Whisper → add as task title. Needs `OPENAI_API_KEY` in env.

- **Sunday retrospective + capacity learning.** Show planned vs done; track rolling 4-week average completion rate; suggest next week's budget.
- **Priorities / effort tags.** `#p1`, `~30m` in the task title; bot parses and displays.
- **Carry-over automation.** Unchecked items Sunday → back to backlog or auto-assigned to same day next week.
- **Inline backlog assignment from anywhere.** When you add a task, immediately ask "which day?"
- **Weekly export.** `/export` dumps the week as markdown or CSV.
- **Web dashboard.** Read-only at first; a small Go HTTP handler serving your data as a calendar view.

---

## Rough time budget

If you're comfortable in Go and work on it in focused 2-hour sessions: **Phases 0–5 are ~4–6 sessions** (8–12 hours of actual coding). The planning flow (Phase 4) is the single biggest chunk because of the stateful conversation.

## Immediate next step

Phase 0: create the bot with BotFather and get `/ping` replying. Once that's working, Phase 1 is the first real milestone.

When you're ready to start writing code, I can scaffold the repo (module layout, Dockerfile, goose setup, initial migration, and a working `/start` + `/ping` handler) as the Phase 0 deliverable.
