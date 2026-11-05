#!/bin/bash
# FSM native launcher (no Docker). Data lives next to this script.
cd "$(dirname "$(readlink -f "$0")")" || exit 1
CONF=./start.conf
need() { command -v "$1" >/dev/null 2>&1; }

# 1. dependencies
MISSING=""
for c in xz tar; do need "$c" || MISSING="$MISSING $c"; done
[ -n "$(ls /etc/ssl/certs 2>/dev/null)" ] || MISSING="$MISSING ca-certificates"
if [ -n "$MISSING" ]; then
    echo "Missing:$MISSING"
    if need apt-get && [ "$(id -u)" = "0" ]; then
        read -p "Install with apt-get now? [Y/n]: " a
        case "$a" in n|N) echo "Install them and run again."; exit 1;; esac
        apt-get update && apt-get install -y xz-utils tar ca-certificates || exit 1
    else
        echo "Install xz, tar and CA certificates with your package manager, then run again."
        exit 1
    fi
fi

# 2. conf.json
if [ ! -f ./conf.json ]; then
    echo "conf.json is missing next to start.sh. Extract the archive again."
    exit 1
fi

local_ip() {
    ip route get 1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src") {print $(i+1); exit}}'
}

run_setup() {
    echo "=== FSM Setup ==="
    echo ""
    while :; do
        read -p "Web UI port [8080]: " PORT
        PORT=${PORT:-8080}
        case "$PORT" in ''|*[!0-9]*) echo "  numbers only";; *) [ "$PORT" -ge 1 ] && [ "$PORT" -le 65535 ] && break; echo "  1-65535";; esac
    done
    DEF_IP=$(local_ip); DEF_IP=${DEF_IP:-localhost}
    read -p "IP shown to players and in the banner [$DEF_IP]: " DISPLAY_IP
    DISPLAY_IP=${DISPLAY_IP:-$DEF_IP}
    printf 'PORT=%s\nDISPLAY_IP=%s\n' "$PORT" "$DISPLAY_IP" > "$CONF"
    echo "Saved start.conf"
    echo ""
}

# 3. first run or launcher mode
if [ ! -f "$CONF" ]; then
    run_setup
elif [ -t 0 ]; then
    echo "Press 'r' to reconfigure or any other key to start..."
    for i in 3 2 1; do
        printf "\r  Starting in [%d]... " "$i"
        if read -t 1 -n 1 key; then
            echo ""
            if [ "$key" = "r" ] || [ "$key" = "R" ]; then rm -f "$CONF"; run_setup; fi
            break
        fi
    done
    echo ""
fi

PORT=$(sed -n 's/^PORT=//p' "$CONF"); PORT=${PORT:-8080}
DISPLAY_IP=$(sed -n 's/^DISPLAY_IP=//p' "$CONF"); DISPLAY_IP=${DISPLAY_IP:-$(local_ip)}

# 4. port check
if need ss && ss -ltn "sport = :$PORT" 2>/dev/null | grep -q LISTEN; then
    echo "Port $PORT is already in use. Is another FSM running? Reconfigure with R at start."
    exit 1
fi

# update hint (silent when offline)
if [ -f .fsm-version ] && need curl; then
    LATEST=$(curl -fsS -m 3 https://api.github.com/repos/joey00797-cell/factorio-server-manager/releases/latest 2>/dev/null | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
    CURV=$(cat .fsm-version)
    if [ -n "$LATEST" ] && [ "$LATEST" != "$CURV" ]; then
        echo "Update available: $LATEST (installed $CURV). Stop FSM and run:"
        echo "  curl -fsSLO https://raw.githubusercontent.com/joey00797-cell/factorio-server-manager/master/install.sh && FSM_DIR=$PWD bash install.sh"
        echo ""
    fi
fi

# 5. run
export FSM_SERVER_IP="$DISPLAY_IP"
export FSM_BANNER=1
exec ./factorio-server-manager \
    --port "$PORT" \
    --servers-root "$PWD/factorio-server" \
    --fsm-data-dir "$PWD/fsm-data" \
    --locales-dir "$PWD/fsm-data/locales" \
    --autostart false
