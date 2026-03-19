# neukeiho-agent

**neukeiho-agent** is the lightweight node agent for [NeuKeiho](https://github.com/meomkarjagtap/neukeiho).  
It runs as a systemd service on each monitored Linux VM, collects system metrics from `/proc`, and pushes them to the NeuKeiho controller over HTTP.

---

## Metrics Collected

| Metric | Source | Unit |
|---|---|---|
| CPU usage | `/proc/stat` | % |
| Memory usage | `/proc/meminfo` | % |
| Disk usage | `syscall.Statfs` on `/` | % |
| Network Rx | `/proc/net/dev` | Mbps |
| Network Tx | `/proc/net/dev` | Mbps |

---

## Installation

### Build from source

```bash
git clone https://github.com/meomkarjagtap/neukeiho-agent
cd neukeiho-agent
go build -o neukeiho-agent ./cmd/neukeiho-agent
sudo mv neukeiho-agent /usr/bin/neukeiho-agent
```

### Deploy via NeuKeiho (recommended)

The recommended way to deploy agents is via the NeuKeiho controller:

```bash
neukeiho deploy
```

This runs the bundled Ansible playbook which installs the binary, writes config, and sets up the systemd service automatically on all nodes defined in `neukeiho.toml`.

---

## Manual Setup

### 1. Create config directory

```bash
sudo mkdir -p /etc/neukeiho-agent
sudo mkdir -p /var/log/neukeiho-agent
```

### 2. Write config

```bash
sudo cp config/agent.conf.example /etc/neukeiho-agent/agent.conf
sudo vim /etc/neukeiho-agent/agent.conf
```

```ini
[agent]
node_id           = web-01
controller_host   = 192.168.1.100
controller_port   = 9100
push_interval     = 10
log_path          = /var/log/neukeiho-agent/agent.log
```

### 3. Create systemd service

```ini
# /etc/systemd/system/neukeiho-agent.service
[Unit]
Description=NeuKeiho Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/neukeiho-agent --config /etc/neukeiho-agent/agent.conf
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now neukeiho-agent
```

### 4. Verify

```bash
sudo systemctl status neukeiho-agent
sudo tail -f /var/log/neukeiho-agent/agent.log
```

---

## Config Reference

| Key | Default | Description |
|---|---|---|
| `node_id` | (required) | Unique identifier for this node |
| `controller_host` | (required) | NeuKeiho controller IP or hostname |
| `controller_port` | `9100` | NeuKeiho controller port |
| `push_interval` | `10` | How often to push metrics (seconds) |
| `log_path` | `/var/log/neukeiho-agent/agent.log` | Log file path |

---

## How It Works

Every `push_interval` seconds the agent:

1. Reads `/proc/stat` → calculates CPU % since last sample
2. Reads `/proc/meminfo` → calculates memory used %
3. Calls `syscall.Statfs("/")` → calculates disk used %
4. Reads `/proc/net/dev` → calculates Rx/Tx Mbps since last sample
5. HTTP POSTs a JSON payload to `http://<controller_host>:<controller_port>/metrics`

```json
{
  "node_id":        "web-01",
  "timestamp":      "2026-03-19T10:00:00Z",
  "cpu_percent":    42.3,
  "memory_percent": 67.1,
  "disk_percent":   55.8,
  "network_rx_mbps": 12.4,
  "network_tx_mbps": 3.1
}
```

---

## Requirements

- Linux (reads `/proc` directly)
- Go 1.22+ (to build from source)
- Network access to NeuKeiho controller on configured port

---

## Project Structure

```
neukeiho-agent/
├── cmd/neukeiho-agent/        # Agent entrypoint
├── internal/
│   ├── metrics/               # /proc readers (CPU, memory, disk, network)
│   └── reporter/              # HTTP push to controller
├── config/
│   └── agent.conf.example
└── README.md
```

---

## Related

- [neukeiho](https://github.com/meomkarjagtap/neukeiho) — the controller
- [NeuRader](https://github.com/neurader/neurader) — Ansible execution observability

---

## License

MIT
