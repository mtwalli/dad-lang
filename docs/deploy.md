# Putting a program online

`dad serve` runs a program as a real web server. Everyone who visits sees the same tables,
which live in one SQLite file on the server.

```sh
dad serve todo.dad                        # or: ضاد انشر todo.dad
dad serve todo.dad --port 3000 --data /var/lib/todo/todo.db
```

It is the server of [`dad dev`](server.md#dad-dev), minus what only makes sense on your own
computer:

| | `dad dev` | `dad serve` |
|---|---|---|
| Listens on | your computer only (`127.0.0.1`) | every network interface (`0.0.0.0`); `--host` to change it |
| Port | `--port`, default 8080 | `--port`, or `$PORT` as hosting platforms set it, default 8080 |
| Developer page `/_dad` | yes: functions, routes, and every table's rows | **no**: `/_dad/…` answers 404, except what pages need |
| Reloads when files change | yes | no: restart it to update the program |
| A program with mistakes | keeps the last good version running | doesn't start (exit code 1) |
| Logs | the program's output | the program's output, plus one line per request on stderr |
| Stops | Ctrl+C | Ctrl+C or `SIGTERM`: it finishes the requests in progress and closes the database |

The site serves:

- your [interface](ui.md) at `/`, when the program has one;
- your [routes](server.md#routes) (`مسار`);
- your server functions at `POST /_dad/call/<name>`;
- `/_dad/health`, which answers `{"ok": true}` for health checks.

## Where the data is kept

By default the data goes in `.dad/<name>.db` next to the program, as with `dad dev`. On a
server, keep it somewhere that survives redeploys and point `--data` there:

```sh
dad serve app.dad --data /data/app.db
```

Changing a `جدول` changes the database the next time the program starts, the same way
`dad dev` does: see [Changing a table](server.md#changing-a-table). A removed field's data is
kept, hidden. A type change that some records can't take stops the program from starting,
and the error lists those records.

To back up while the server runs, use SQLite's own backup, not a plain copy:

```sh
sqlite3 /data/app.db ".backup /backups/app-$(date +%F).db"
```

## HTTPS

`dad serve` speaks plain HTTP. For a public site, put it behind a server that handles
certificates. With [Caddy](https://caddyserver.com), this `Caddyfile` is enough:

```text
example.com {
    reverse_proxy localhost:8080
}
```

Then run `dad serve app.dad --host 127.0.0.1`, so only Caddy can reach it.

## A Linux server with systemd

```ini title="/etc/systemd/system/todo.service"
[Unit]
Description=To-do app in Ḍād
After=network.target

[Service]
WorkingDirectory=/srv/todo
ExecStart=/usr/local/bin/dad serve todo.dad --host 127.0.0.1 --data /var/lib/todo/todo.db
Restart=on-failure
DynamicUser=yes
StateDirectory=todo

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemctl enable --now todo
journalctl -u todo -f        # the request log
```

## Docker, and platforms such as Fly.io or Render

`dad` is a single program with no dependencies. SQLite is built in, so a small image is
enough:

```dockerfile title="Dockerfile"
FROM golang:1.26 AS dad
RUN CGO_ENABLED=0 go install github.com/mtwalli/dad-lang/cmd/dad@latest

FROM gcr.io/distroless/static
COPY --from=dad /go/bin/dad /dad
COPY . /app
WORKDIR /app
VOLUME /data
CMD ["/dad", "serve", "todo.dad", "--data", "/data/todo.db"]
```

The platform sets `$PORT`, and `dad serve` uses it. Give the app a persistent volume
mounted at `/data`, so the database outlives each deploy. Run a single copy of the app: see
the next section.

## What to know before you rely on it

- **One copy.** The data is in one SQLite file, and pages keep their state in the
  server's memory. Run one `dad serve`, not several behind a load balancer.
- **One request at a time.** Requests are handled in turn. This is plenty for a small team
  or a class, but not for heavy traffic.
- **A restart reloads open pages.** What a visitor had typed but not saved is lost. The
  tables are not lost.
- **No sign-in yet.** Every visitor can call every server function and route. Don't put a
  program online with data that only some people should see or change. Users and
  permissions are on the [roadmap](ROADMAP.md).
