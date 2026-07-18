# Running FSM (develop branch)

This branch uses a custom two-stage Docker build and bind mounts instead of named volumes.

## Prerequisites

- Docker
- Python 3
- Linux host (tested on Ubuntu/Debian)

## Quick start

### 1. Clone

```bash
git clone -b develop https://github.com/joey00797-cell/factorio-server-manager.git
cd factorio-server-manager
```

### 2. Create data directories

```bash
mkdir -p /opt/fsm-data /opt/factorio-server
```

### 3. Create run Dockerfile

```bash
cat > /tmp/Dockerfile-run << 'DOCKERFILE'
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates xz-utils curl jq && rm -rf /var/lib/apt/lists/*
COPY entrypoint.sh /opt/entrypoint.sh
COPY factorio-server-manager /opt/fsm/factorio-server-manager
COPY app /opt/fsm/app
COPY conf.json /opt/fsm/conf.json
RUN chmod +x /opt/entrypoint.sh /opt/fsm/factorio-server-manager
EXPOSE 80
ENTRYPOINT ["/opt/entrypoint.sh"]
DOCKERFILE
```

### 4. Add build alias to ~/.bashrc

```bash
cat >> ~/.bashrc << 'ALIAS'
fsm-build() {
    cd ~/factorio-server-manager
    docker stop ofsm 2>/dev/null; docker rm ofsm 2>/dev/null
    rm -rf /tmp/fsm-output2
    docker build -f docker/Dockerfile-build --target build -t fsm-build-stage . && \
    docker create --name fsm-extract fsm-build-stage && \
    docker cp fsm-extract:/go/src/factorio-server-manager/build/. /tmp/fsm-output2/ && \
    docker rm fsm-extract && \
    python3 -c "
import zipfile, os
z = zipfile.ZipFile('/tmp/fsm-output2/factorio-server-manager-linux.zip')
for member in z.infolist():
    # strip leading 'factorio-server-manager/' from paths
    parts = member.filename.split('/', 1)
    if len(parts) < 2 or not parts[1]:
        continue
    member.filename = parts[1]
    z.extract(member, '/tmp/fsm-output2/')
" && \
    cp ~/factorio-server-manager/docker/entrypoint.sh /tmp/fsm-output2/ && \
    docker build -f /tmp/Dockerfile-run -t my-fsm:latest /tmp/fsm-output2/ && \
    docker run -d \
        --name ofsm \
        -p 80:80 \
        -p 34197-34207:34197-34207/udp \
        -v /opt/fsm-data:/opt/fsm-data \
        -v /opt/factorio-server:/opt/factorio-server \
        my-fsm:latest && \
    echo "FSM READY"
}
ALIAS
source ~/.bashrc
```

### 5. Build and run

```bash
fsm-build
```

Default admin credentials are printed to docker logs on first run:

```bash
docker logs ofsm 2>&1 | grep -i "password\|username"
```

## Notes

- Port range `34197-34207/udp` — one UDP port per Factorio server instance
- `/opt/fsm-data` — FSM config, SQLite DB, custom locales
- `/opt/factorio-server` — shared Factorio version cache + per-server instances
- No Factorio binary pre-installed — download stable/experimental directly from UI
- After first login, change admin password in User Management

## Directory structure (auto-created on first run)

```
/opt/factorio-server/
├── versions/          # Factorio binaries (shared between servers)
├── downloads/         # Cached .tar.xz archives
└── instances/
    ├── 1/             # Server 1
    │   ├── saves/
    │   ├── mods/
    │   └── config/
    └── 2/             # Server 2
        └── ...
```

### 6. Clean

```bash
cat >> ~/.bashrc << 'ALIAS'
fsm-clean() {
    echo "=== FSM cleanup ==="

    for name in ofsm fsm-extract; do
        docker stop $name 2>/dev/null && echo "stopped: $name" || true
        docker rm $name 2>/dev/null && echo "removed container: $name" || true
    done

    for img in my-fsm fsm-build-stage; do
        docker rmi $img 2>/dev/null && echo "removed image: $img" || true
    done

    rm -rf /tmp/fsm-output2
    rm -f /tmp/Dockerfile-run
    echo "removed: /tmp/fsm-output2, /tmp/Dockerfile-run"

    read -p "Удалить /opt/fsm-data и /opt/factorio-server? [y/N] " confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        rm -rf /opt/fsm-data /opt/factorio-server
        echo "removed: /opt/fsm-data, /opt/factorio-server"
    else
        echo "skipped: data directories"
    fi

    echo "=== done ==="
}
ALIAS
source ~/.bashrc
```

### 7. Run clean
 
```bash
fsm-clean
```
---

## 🎁 Bonus: One-liner setup

If you read this far — here's your reward:

```bash
git clone -b develop https://github.com/joey00797-cell/factorio-server-manager.git ~/factorio-server-manager && cd ~/factorio-server-manager && bash setup.sh
```

This will clone the repo, ask a few questions, and set everything up automatically.
