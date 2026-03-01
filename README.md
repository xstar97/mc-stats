# mc-stats

**Custom Docker image for [MinecraftStats](https://github.com/pdinklag/MinecraftStats)**  

A containerized MinecraftStats CLI and web server with support for multiple instances, automated stats generation, and configurable intervals.

---

## Features

- Run **one or multiple MinecraftStats instances** in a single container  
- Automatic periodic **stats generation**  
- **Per-instance configurable intervals**  
- Serves each instance’s **web stats directory**  
- **JSON `/status` endpoint** for monitoring generation status  
- Lightweight **Alpine-based Docker image**  

---

## Quick Start

Check out the examples directory for the docker-compose.yaml and config.json

```yaml
services:
  mc-stats:
    image: ghcr.io/xstar97/mc-stats:3.3.1
    container_name: mc-stats
    restart: unless-stopped
    ports:
      - 8080:8080 # internal port can be set via PORT variable
    environment:
      PORT: 8080
      # global
      INTERVAL_SECONDS: 300
      INSTANCES=survival # comma delim instance separator
      # separate interval updater per instance
      # INTERVAL_SECONDS_SURVIVAL=600
    volumes:
      # do whatever you want here; internal path should be /config for the data and the instance public dir(web) in /config
      - ./config:/config
      # mount minecraft data here
```

### Single or Multi-Instance Setup

- **Single instance:** Mount your config directory and start the container.
- **Multiple instances:** Set `INSTANCES` to a comma-separated list of instance names in your Docker Compose or environment variables.  
  - Each instance requires its **own `config.json`** and internal web directory.  
  - Example: `INSTANCES=survival,creative`  

### Notes on Configuration

- The container expects all paths to remain under `/config`.  
- You can rename instance directories as you like, but each instance must have:
    - /config/<instance_name>/config.json
    - /config/<instance_name>/public
    - /config/<instance_name>/events
    - /config/<instance_name>/stats

- To serve stats for multiple Minecraft servers separately, mount each server’s data into an internal path accessible by the container (e.g., `/data/server1`) and update `server.sources.path` in the respective `config.json`.

---

## Example `config.json`

This is a sample configuration for a single instance. Replace `server1` with your instance directory name of your choosing.

```json
{
"client": {
  "defaultLanguage": "en",
  "playerCacheUUIDPrefix": 2,
  "playersPerPage": 100,
  "serverName": null,
  "showLastOnline": true
},
"crown": {
  "bronze": 1,
  "gold": 4,
  "silver": 2
},
"data": {
  "documentRoot": "/config/server1/public",
  "eventsDir": "/config/server1/events",
  "statsDir": "/config/server1/stats"
},
"players": {
  "excludeBanned": true,
  "excludeOps": false,
  "excludeUUIDs": [],
  "inactiveDays": 7,
  "minPlaytime": 60,
  "profileUpdateInterval": 3,
  "updateInactive": false
},
"server": {
  "sources": [
    {
      "path": "/data/server1",
      "worldName": "world"
    }
  ]
}
}
```

⚠️ Make sure all paths inside config.json match your mounted directories.
The documentRoot, eventsDir, and statsDir must always point inside /config/<instance_name>/.

## Environment Variables

| Variable                  | Default | Description                                  |
|---------------------------|---------|----------------------------------------------|
| `PORT`                    | 8080    | HTTP server port                             |
| `INTERVAL_SECONDS`        | 300     | Default stats generation interval (seconds) |
| `INSTANCES`               | -       | Comma-separated list of instance names      |
| `INTERVAL_SECONDS_<INSTANCE>` | -   | Optional per-instance interval override     |

Accessing the Web UI

Single instance: http://localhost:8080 → serves the instance directly

Multiple instances: http://localhost:8080 → landing page listing all instances

Status Endpoint

Check generation status of all instances:

GET /status

Returns JSON:

```
[
  {
    "name": "survival",
    "last_generated": "2026-02-28T20:39:19Z",
    "last_duration": 5.032,
    "success": true,
    "current_pid": 0,
    "time_until_next_run": 295
  },
  {
    "name": "creative",
    "last_generated": "2026-02-28T20:34:19Z",
    "last_duration": 4.987,
    "success": true,
    "current_pid": 0,
    "time_until_next_run": 596
  }
]
```
