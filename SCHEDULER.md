# 🕐 Scheduler - Automated Task Execution

The scheduler is a powerful feature that allows you to automate router management by running tasks at specified intervals or fixed times. Perfect for keeping routes and DNS records up-to-date automatically without manual intervention.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Execution Modes](#execution-modes)
- [Retry Mechanism](#retry-mechanism)
- [Task Queue](#task-queue)
- [Examples](#examples)
- [Tips](#tips)

---

## Overview

The scheduler runs as a long-lived process that continuously monitors configured tasks and executes them according to their schedules. It's designed to be reliable, efficient, and easy to configure.

**Use cases:**
- Automatically update routes from dynamic sources every few hours
- Refresh DNS records at specific times daily
- Perform maintenance tasks during off-peak hours
- Manage multiple routers with a single scheduler instance

---

## Key Features

### ⏰ Flexible Scheduling

- **Interval-based**: Run tasks every N hours/minutes (e.g., `"3h"`, `"30m"`)
- **Time-based**: Run tasks at specific times (e.g., `["06:00", "12:00", "18:00"]`)

### 🔗 Command Chaining

Execute multiple commands sequentially in a single task:
```yaml
commands:
  - delete-routes  # First delete old routes
  - add-routes     # Then add new routes
```

### 🔄 Retry Mechanism

Automatically retry failed tasks with configurable attempts and delays:
```yaml
retry: 3           # Retry up to 3 times
retryDelay: "30s"  # Wait 30 seconds between retries
```

### 🎯 Multi-Router Support

Manage multiple routers with a single task:
```yaml
configs:
  - /path/to/router1.yaml
  - /path/to/router2.yaml
  - /path/to/router3.yaml
```

### 📋 Sequential Execution

Tasks are executed in a FIFO queue, ensuring no conflicts when multiple tasks trigger simultaneously.

---

## Quick Start

### 1. Create Scheduler Configuration

Create a `scheduler.yaml` file:

```yaml
tasks:
  - name: "Update routes every 3 hours"
    commands:
      - add-routes
    configs:
      - /path/to/router1.yaml
    interval: "3h"
```

### 2. Run Scheduler

```bash
./gokeenapi scheduler --config scheduler.yaml
```

### 3. Monitor Execution

The scheduler will display:
- Task schedules on startup
- Execution progress
- Success/failure status
- Retry attempts (if configured)

---

## Configuration

### Task Structure

```yaml
tasks:
  - name: "Task name"              # Required: Descriptive name
    commands:                       # Required: List of commands to execute
      - command1
      - command2
    configs:                        # Required: List of router configs
      - /path/to/config1.yaml
      - /path/to/config2.yaml
    interval: "3h"                  # Optional: Execution interval
    times:                          # Optional: Execution times (24h format)
      - "06:00"
      - "12:00"
    retry: 3                        # Optional: Number of retry attempts (default: 0)
    retryDelay: "30s"               # Optional: Delay between retries (default: "1m")
```

### Field Descriptions

| Field        | Type   | Required | Description                                         |
|--------------|--------|----------|-----------------------------------------------------|
| `name`       | string | Yes      | Descriptive name for the task                       |
| `commands`   | array  | Yes      | List of gokeenapi commands to execute               |
| `configs`    | array  | Yes      | List of router configuration file paths             |
| `interval`   | string | No*      | Execution interval (e.g., "30m", "1h", "3h")        |
| `times`      | array  | No*      | Execution times in HH:MM format                     |
| `retry`      | int    | No       | Number of retry attempts on failure (≥ 0)           |
| `retryDelay` | string | No       | Delay between retries (≥ 1s, default: "1m")         |

*Either `interval` or `times` must be specified, but not both.

### Validation Rules

- `interval` must be ≥ 1 second
- `times` must be in 24-hour format (HH:MM)
- `retry` must be ≥ 0
- `retryDelay` must be ≥ 1 second
- Cannot use both `interval` and `times` in the same task

---

## Execution Modes

### Interval-Based Execution

Run tasks periodically at fixed intervals:

```yaml
- name: "Update routes every 3 hours"
  commands:
    - add-routes
  configs:
    - /path/to/router.yaml
  interval: "3h"
```

**Supported formats:**
- `"30s"` - 30 seconds
- `"5m"` - 5 minutes
- `"1h"` - 1 hour
- `"2h30m"` - 2 hours 30 minutes
- `"24h"` - 24 hours

**Behavior:**
- First execution happens immediately on scheduler start
- Subsequent executions occur at the specified interval
- Timer resets after each execution completes

### Time-Based Execution

Run tasks at specific times of day:

```yaml
- name: "Update DNS at fixed times"
  commands:
    - add-dns-records
  configs:
    - /path/to/router.yaml
  times:
    - "06:00"  # 6 AM
    - "12:00"  # 12 PM
    - "18:00"  # 6 PM
```

**Behavior:**
- Executes at each specified time
- If all times have passed today, waits until earliest time tomorrow
- Uses 24-hour format (HH:MM)
- Automatically handles timezone

---

## Retry Mechanism

The retry mechanism helps handle transient failures like network issues or router unavailability.

### Configuration

```yaml
- name: "Update routes with retry"
  commands:
    - add-routes
  configs:
    - /path/to/router.yaml
  interval: "1h"
  retry: 3           # Total attempts = 1 + 3 = 4
  retryDelay: "30s"  # Wait 30 seconds between attempts
```

### Behavior

1. **Initial Attempt**: Command executes normally
2. **On Failure**: If command fails, wait `retryDelay` and retry
3. **Retry Attempts**: Continue up to `retry` times
4. **Success**: Stop retrying on first success
5. **Final Failure**: After all attempts exhausted, task fails

### Example Output

```
⌛   Executing add-routes ... 
⛔   Executing add-routes failed after 2.5s
  ▪ Error: exit status 1, connection refused
  ▪ Retrying in 30s...
⌛   Executing add-routes (attempt 2/4) ...
✅   Executing add-routes (attempt 2/4) completed after 1.2s
```

---

## Task Queue

Tasks are executed sequentially in a FIFO (First-In-First-Out) queue.

### Why Sequential?

- **Prevents conflicts**: Multiple tasks modifying the same router simultaneously
- **Resource management**: Avoids overwhelming the router with concurrent requests
- **Predictable behavior**: Tasks execute in a deterministic order

### How It Works

```
Task 1 triggers at 06:00 → Added to queue
Task 2 triggers at 06:00 → Added to queue (waits for Task 1)
Task 3 triggers at 06:01 → Added to queue (waits for Task 2)

Execution order: Task 1 → Task 2 → Task 3
```

### Implications

- If a task takes long to execute, subsequent tasks will wait
- Use appropriate intervals to avoid queue buildup
- Monitor execution times to optimize schedules

---

## Examples

### Example 1: Simple Periodic Update

Update routes (without deleting old) every 3 hours on multiple routers:

```yaml
tasks:
  - name: "Update routes every 3 hours"
    commands:
      - add-routes
    configs:
      - /path/to/router1.yaml
      - /path/to/router2.yaml
      - /path/to/router3.yaml
    interval: "3h"
```

### Example 2: Daily Maintenance with Retry

Delete old and add new routes daily at 2 AM with retry on failure:

```yaml
tasks:
  - name: "Daily route refresh"
    commands:
      - delete-routes
      - add-routes
    configs:
      - /path/to/router.yaml
    times:
      - "02:00"
    retry: 3
    retryDelay: "1m"
```

### Example 3: Multiple Time Windows

Update DNS at multiple times throughout the day:

```yaml
tasks:
  - name: "DNS updates"
    commands:
      - add-dns-records
    configs:
      - /path/to/router.yaml
    times:
      - "06:00"
      - "12:00"
      - "18:00"
      - "23:00"
```

### Example 4: Complex Multi-Router Setup

```yaml
tasks:
  # Frequent updates for critical routers
  - name: "Critical routers - hourly"
    commands:
      - add-routes
    configs:
      - /path/to/critical1.yaml
      - /path/to/critical2.yaml
    interval: "1h"
    retry: 2
    retryDelay: "30s"

  # Less frequent updates for secondary routers
  - name: "Secondary routers - every 6 hours"
    commands:
      - add-routes
    configs:
      - /path/to/secondary1.yaml
      - /path/to/secondary2.yaml
    interval: "6h"
```

---

## Tips

### 1. Choose Appropriate Intervals

- **Too frequent**: Wastes resources, may overwhelm router
- **Too infrequent**: Data may become stale
- **Recommended**: Start with 3-6 hours, adjust based on needs

### 2. Use Retry Wisely

- Enable retry for network-dependent operations
- Set reasonable `retryDelay` (30s - 2m)
- Don't exceed 3-5 retry attempts

### 3. Schedule Maintenance During Off-Peak

```yaml
times:
  - "02:00"  # Low traffic time
```

### 4. Test Configuration First

```bash
# Test individual commands before scheduling
./gokeenapi add-routes --config router.yaml

# Then add to scheduler
```

---

## Running as a Service

### systemd (Linux)

Create `/etc/systemd/system/gokeenapi-scheduler.service`:

```ini
[Unit]
Description=Gokeenapi Scheduler
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/gokeenapi
ExecStart=/path/to/gokeenapi scheduler --config /path/to/scheduler.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable gokeenapi-scheduler
sudo systemctl start gokeenapi-scheduler
sudo systemctl status gokeenapi-scheduler
```

### Docker

```bash
docker run -d \
  --name gokeenapi-scheduler \
  --restart unless-stopped \
  -v /path/to/scheduler.yaml:/scheduler.yaml \
  -v /path/to/configs:/configs \
  noksa/gokeenapi:stable \
  scheduler --config /scheduler.yaml
```

---

## See Also

- [Main README](README.md) - General documentation
- [scheduler_example.yaml](scheduler_example.yaml) - Configuration examples
- [Configuration Guide](README.md#configuration) - Router configuration

---

**Need help?** [Open an issue](https://github.com/Noksa/gokeenapi/issues) on GitHub.
