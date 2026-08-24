#!/bin/bash
SETUP_CONF="/opt/fsm-data/setup.conf"
LOG_FILE="/tmp/fsm-boot.log"
CREDS_FILE="/opt/fsm-data/admin-credentials.txt"
IMAGE="${FSM_IMAGE:-ghcr.io/joey00797-cell/fsm:latest}"

run_setup() {
    echo "=== FSM Setup ==="
    echo ""
    read -p "Host IP (your server IP): " HOST_IP < /dev/tty
    HOST_IP=$(echo "$HOST_IP" | sed 's|https\?://||g' | tr -d '/')
    while [ -z "$HOST_IP" ]; do
        echo "  Host IP is required!"
        read -p "Host IP (your server IP): " HOST_IP < /dev/tty
        HOST_IP=$(echo "$HOST_IP" | sed 's|https\?://||g' | tr -d '/')
    done
    read -p "HTTP port [80]: " HTTP_PORT < /dev/tty
    HTTP_PORT=${HTTP_PORT:-80}
    read -p "UDP port start [34197]: " UDP_START < /dev/tty
    UDP_START=${UDP_START:-34197}
    read -p "Number of server slots [10]: " UDP_SLOTS < /dev/tty
    UDP_SLOTS=${UDP_SLOTS:-10}
    UDP_END=$((UDP_START + UDP_SLOTS - 1))

    # macvlan
    USE_MACVLAN=""
    CONTAINER_IP=""
    echo ""
    read -p "Enable macvlan for LAN broadcast? [y/N]: " USE_MACVLAN < /dev/tty
    if [ "$USE_MACVLAN" = "y" ] || [ "$USE_MACVLAN" = "Y" ]; then
        # Suggest HOST_IP + 1
        BASE=$(echo $HOST_IP | cut -d. -f1-3)
        LAST=$(echo $HOST_IP | cut -d. -f4)
        SUGGESTED_IP="$BASE.$((LAST + 1))"
        echo "  Reserve this IP on your router/DHCP!"
        read -p "  Container IP [$SUGGESTED_IP]: " CONTAINER_IP < /dev/tty
        CONTAINER_IP=${CONTAINER_IP:-$SUGGESTED_IP}

        # Derive subnet and gateway from HOST_IP
        BASE=$(echo $HOST_IP | cut -d. -f1-3)
        IFACE=$(ip route | awk '/default/ {print $5; exit}')
        SUBNET="${BASE}.0/24"
        GATEWAY="${BASE}.1"
        read -p "  Subnet [$SUBNET]: " INPUT_SUBNET < /dev/tty
        SUBNET=${INPUT_SUBNET:-$SUBNET}
        read -p "  Gateway [$GATEWAY]: " INPUT_GW < /dev/tty
        GATEWAY=${INPUT_GW:-$GATEWAY}
    fi

    mkdir -p /opt/fsm-data
    cat > $SETUP_CONF << CONF
HOST_IP=${HOST_IP}
HTTP_PORT=${HTTP_PORT}
UDP_START=${UDP_START}
UDP_END=${UDP_END}
USE_MACVLAN=${USE_MACVLAN}
CONTAINER_IP=${CONTAINER_IP}
IFACE=${IFACE}
SUBNET=${SUBNET}
GATEWAY=${GATEWAY}
CONF

    # Generate fsm-start.sh
    if [ "$USE_MACVLAN" = "y" ] || [ "$USE_MACVLAN" = "Y" ]; then
        cat > /opt/fsm-data/fsm-start.sh << SCRIPT
#!/bin/bash
# pre-flight checks
if ! docker info > /dev/null 2>&1; then
    echo "ERROR: Docker is not running!"
    exit 1
fi
if [ ! -w "/opt/fsm-data" ]; then
    echo "ERROR: Cannot write to /opt/fsm-data — check permissions (chmod 777 /opt/fsm-data)"
    exit 1
fi
if [ ! -f "/opt/fsm-data/setup.conf" ]; then
    echo "ERROR: setup.conf not found — run setup first:"
    echo "  docker run -it --rm -v /opt/fsm-data:/opt/fsm-data ${IMAGE}"
    exit 1
fi
echo "Pre-flight OK"
docker stop fsm 2>/dev/null; docker rm fsm 2>/dev/null || true
docker network rm fsm-lan 2>/dev/null || true
docker network create -d macvlan \\
    --subnet=${SUBNET} \\
    --gateway=${GATEWAY} \\
    -o parent=${IFACE} \\
    fsm-lan
docker run -d \\
    --name fsm \\
    --network fsm-lan \\
    --ip ${CONTAINER_IP} \\
    -e FSM_SERVER_IP=${CONTAINER_IP} \\
    -v /opt/fsm-data:/opt/fsm-data \\
    -v /opt/factorio-server:/opt/factorio-server \\
    ${IMAGE}
SCRIPT
    else
        cat > /opt/fsm-data/fsm-start.sh << SCRIPT
#!/bin/bash
# pre-flight checks
if ! docker info > /dev/null 2>&1; then
    echo "ERROR: Docker is not running!"
    exit 1
fi
if [ ! -w "/opt/fsm-data" ]; then
    echo "ERROR: Cannot write to /opt/fsm-data — check permissions (chmod 777 /opt/fsm-data)"
    exit 1
fi
if [ ! -f "/opt/fsm-data/setup.conf" ]; then
    echo "ERROR: setup.conf not found — run setup first:"
    echo "  docker run -it --rm -v /opt/fsm-data:/opt/fsm-data ${IMAGE}"
    exit 1
fi
echo "Pre-flight OK"
docker stop fsm 2>/dev/null; docker rm fsm 2>/dev/null || true
docker run -d \\
    --name fsm \\
    -p ${HTTP_PORT}:80 \\
    -p ${UDP_START}-${UDP_END}:${UDP_START}-${UDP_END}/udp \\
    -e FSM_SERVER_IP=${HOST_IP} \\
    -v /opt/fsm-data:/opt/fsm-data \\
    -v /opt/factorio-server:/opt/factorio-server \\
    ${IMAGE}
SCRIPT
    fi

    cat >> /opt/fsm-data/fsm-start.sh << 'SCRIPT2'
echo "Waiting for FSM..."
for i in $(seq 1 15); do
    if docker logs fsm 2>&1 | grep -q "FSM is Ready"; then
        break
    fi
    sleep 1
done
docker logs fsm 2>&1 | grep -A30 "FSM is Ready" | head -30
SCRIPT2

    chmod +x /opt/fsm-data/fsm-start.sh

    # Generate fsm-start.bat for Windows (no macvlan)
    cat > /opt/fsm-data/fsm-start.bat << BATCH
@echo off
docker stop fsm 2>nul
docker rm fsm 2>nul
docker run -d ^
    --name fsm ^
    -p ${HTTP_PORT}:80 ^
    -p ${UDP_START}-${UDP_END}:${UDP_START}-${UDP_END}/udp ^
    -v C:\\fsm-data:/opt/fsm-data ^
    -v C:\\factorio-server:/opt/factorio-server ^
    ${IMAGE}
echo Waiting for FSM...
timeout /t 8 /nobreak >nul
docker logs fsm
BATCH

    echo ""
    echo "========================================="
    echo "  Setup complete!"
    echo "========================================="
    echo ""
    echo "  Start (Linux/Mac):"
    echo "    bash /opt/fsm-data/fsm-start.sh"
    echo ""
    echo "  Start (Windows):"
    echo "    C:\\fsm-data\\fsm-start.bat"
    echo ""
    echo "  Alias (optional):"
    echo "    echo \"alias fsm='bash /opt/fsm-data/fsm-start.sh'\" >> ~/.bashrc"
    echo "    source ~/.bashrc"
    echo ""
    echo "  Reconfigure:"
    echo "    docker run -it --rm \\"
    echo "      -v /opt/fsm-data:/opt/fsm-data \\"
    echo "      ${IMAGE}"
    echo ""
    echo "========================================="
    echo ""
}

init_conf() {
    mkdir -p /opt/fsm-data
    jq_cmd='.'
    jq_cmd="${jq_cmd} | .sq_lite_database_file = \"/opt/fsm-data/sqlite.db\""
    jq_cmd="${jq_cmd} | .log_file = \"/opt/fsm-data/factorio-server-manager.log\""
    jq "${jq_cmd}" /opt/fsm/conf.json > /opt/fsm-data/conf.json
}

countdown_or_reconfig() {
    echo "Press 'r' to reconfigure or any other key to start..."
    for i in 3 2 1; do
        printf "\r  Starting in [%d]... " $i
        if read -t 1 -n 1 key < /dev/tty 2>/dev/null; then
            echo ""
            if [ "$key" = "r" ] || [ "$key" = "R" ]; then
                echo "Reconfiguring..."
                read -p "Clear all data? [y/N]: " CLEAR < /dev/tty
                rm -f $SETUP_CONF
                if [ "$CLEAR" = "y" ] || [ "$CLEAR" = "Y" ]; then
                    rm -rf /opt/fsm-data/* /opt/factorio-server/*
                    echo "✓ Data cleared"
                fi
                return 1
            fi
            return 0
        fi
    done
    echo ""
    return 0
}

show_ready() {
    local FIRST=$1
    if [ "$FIRST" = "1" ]; then
        for i in $(seq 1 10); do
            grep -q "Default admin password" $LOG_FILE 2>/dev/null && break
            sleep 1
        done
        PASS=$(grep "Default admin password" $LOG_FILE 2>/dev/null | tail -1 | awk '{print $NF}')
        if [ -n "$PASS" ]; then
            echo "$PASS" > $CREDS_FILE
            chmod 600 $CREDS_FILE
        fi
    fi

    SAVED_PASS=$(cat $CREDS_FILE 2>/dev/null)
    DISPLAY_IP=${CONTAINER_IP:-${HOST_IP}}

    echo ""
    echo "========================================"
    echo "  FSM is Ready!"
    echo "========================================"
    printf "  URL:  http://%s:%s\n" "${DISPLAY_IP}" "${HTTP_PORT}"
    printf "  UDP:  %s-%s/udp\n" "${UDP_START}" "${UDP_END}"
    echo "  User: admin"
    if [ -n "$SAVED_PASS" ]; then
        printf "  Pass: %s\n" "${SAVED_PASS}"
        echo "  (initial password, change after login)"
    fi
    echo "----------------------------------------"
    echo "  Reconfigure:"
    echo "    docker run -it --rm \\"
    echo "      -v /opt/fsm-data:/opt/fsm-data \\"
    echo "      ${IMAGE}"
    echo "  Logs: docker logs fsm -f"
    echo "========================================"
    echo ""
}

mkdir -p /opt/fsm-data

if [ ! -w "/opt/fsm-data" ]; then
    echo ""
    echo "========================================"
    echo "  FSM ERROR: Permission denied!"
    echo "========================================"
    echo "  Cannot write to /opt/fsm-data."
    echo "  Fix with: chmod 777 /opt/fsm-data"
    echo "========================================"
    exit 1
fi

if [ ! -f "$SETUP_CONF" ]; then
    if ! [ -t 0 ]; then
        echo ""
        echo "========================================"
        echo "  FSM: First-time setup required!"
        echo "========================================"
        echo "  No setup.conf found and no TTY available."
        echo "  Run the setup wizard first:"
        echo ""
        echo "    docker run -it --rm \\"
        echo "      -v /opt/fsm-data:/opt/fsm-data \\"
        echo "      ghcr.io/joey00797-cell/fsm:latest"
        echo ""
        echo "  Then start FSM normally."
        echo "========================================"
        exit 1
    fi
    run_setup
    exit 0
fi

. $SETUP_CONF
FIRST_RUN=0

if [ -t 0 ]; then
    if ! countdown_or_reconfig; then
        run_setup
        exit 0
    fi
    . $SETUP_CONF
fi

init_conf

printf "Starting FSM"
cd /opt/fsm
./factorio-server-manager \
    --conf /opt/fsm-data/conf.json \
    --dir /opt/factorio-server \
    --servers-root /opt/factorio-server \
    --port ${HTTP_PORT:-80} \
    --autostart false > $LOG_FILE 2>&1 &
FSM_PID=$!

for i in $(seq 1 8); do
    sleep 1
    printf "."
    if grep -q "FSM starting on" $LOG_FILE 2>/dev/null; then
        break
    fi
done
echo " done!"

if grep -q "Created default admin user" $LOG_FILE 2>/dev/null; then
    FIRST_RUN=1
fi

show_ready $FIRST_RUN

tail -f $LOG_FILE &
TAIL_PID=$!
trap "kill $FSM_PID $TAIL_PID 2>/dev/null" INT TERM
wait $FSM_PID
kill $TAIL_PID 2>/dev/null
