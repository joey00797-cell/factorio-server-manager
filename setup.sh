#!/bin/bash


echo "=== FSM Setup ==="
echo ""

# Paths
read -p "FSM data directory [/opt/fsm-data]: " FSM_DATA
FSM_DATA=${FSM_DATA:-/opt/fsm-data}

read -p "Factorio server directory [/opt/factorio-server]: " FACTORIO_DIR
FACTORIO_DIR=${FACTORIO_DIR:-/opt/factorio-server}

read -p "HTTP port [80]: " HTTP_PORT
HTTP_PORT=${HTTP_PORT:-80}

read -p "Factorio UDP port range start [34197]: " UDP_START
UDP_START=${UDP_START:-34197}

read -p "Number of server slots [10]: " UDP_SLOTS
UDP_SLOTS=${UDP_SLOTS:-10}
UDP_END=$((UDP_START + UDP_SLOTS - 1))

# Macvlan setup
echo ""
read -p "Enable LAN broadcast via macvlan? [y/N]: " USE_MACVLAN
CONTAINER_IP=""
if [ "$USE_MACVLAN" = "y" ] || [ "$USE_MACVLAN" = "Y" ]; then
    IFACE=$(ip route | awk '/default/ {print $5; exit}')
    HOST_IP=$(ip -4 addr show $IFACE | awk '/inet / {print $2}' | cut -d/ -f1)
    SUBNET=$(ip -4 addr show $IFACE | awk '/inet / {print $2}' | python3 -c "import sys,ipaddress; n=ipaddress.IPv4Network(sys.stdin.read().strip(),strict=False); print(str(n))")
    GATEWAY=$(ip route | awk '/default/ {print $3; exit}')
    LAST_OCTET=$(echo $HOST_IP | cut -d. -f4)
    BASE=$(echo $HOST_IP | cut -d. -f1-3)
    SUGGESTED_IP="$BASE.$((LAST_OCTET + 1))"
    echo "  Interface: $IFACE | Host IP: $HOST_IP"
    read -p "Container IP [$SUGGESTED_IP]: " CONTAINER_IP
    CONTAINER_IP=${CONTAINER_IP:-$SUGGESTED_IP}
fi

echo ""
echo "=== Configuration ==="
echo "  FSM data:     $FSM_DATA"
echo "  Factorio dir: $FACTORIO_DIR"
echo "  HTTP port:    $HTTP_PORT"
echo "  UDP ports:    $UDP_START-$UDP_END"
if [ -n "$CONTAINER_IP" ]; then
    echo "  macvlan IP:   $CONTAINER_IP"
fi
echo ""
read -p "Proceed? [Y/n]: " confirm
[ "$confirm" = "n" ] || [ "$confirm" = "N" ] && echo "Cancelled." && exit 0

# Create directories
mkdir -p "$FSM_DATA" "$FACTORIO_DIR"
echo "✓ Directories created"

# Save macvlan config if enabled
if [ -n "$CONTAINER_IP" ]; then
    cat > ${FSM_DATA}/macvlan.conf << MACVLAN
IFACE=${IFACE}
SUBNET=${SUBNET}
GATEWAY=${GATEWAY}
CONTAINER_IP=${CONTAINER_IP}
MACVLAN
    echo "✓ macvlan.conf saved"
    echo "⚠ Forward ports 80 and 34197-34206/udp to ${CONTAINER_IP} on your router"
fi

# Create Dockerfile-run
create_dockerfile() {
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
}
create_dockerfile
echo "✓ Dockerfile-run created"

# Build and run function (used both here and in alias)
# Setup macvlan network if config exists
setup_macvlan() {
    if [ -f "${FSM_DATA}/macvlan.conf" ]; then
        . ${FSM_DATA}/macvlan.conf
        docker network rm fsm-lan 2>/dev/null || true
        docker network create -d macvlan \
            --subnet=${SUBNET} \
            --gateway=${GATEWAY} \
            -o parent=${IFACE} \
            fsm-lan
        echo "✓ macvlan network created (${CONTAINER_IP})"
    fi
}

get_network_args() {
    if [ -f "${FSM_DATA}/macvlan.conf" ]; then
        . ${FSM_DATA}/macvlan.conf
        echo "--network fsm-lan --ip ${CONTAINER_IP}"
    else
        echo "-p ${HTTP_PORT}:80 -p ${UDP_START}-${UDP_END}:${UDP_START}-${UDP_END}/udp"
    fi
}

do_build() {
    cd ~/factorio-server-manager
    docker stop ofsm 2>/dev/null; docker rm ofsm 2>/dev/null || true
    docker builder prune -f 2>/dev/null || true
    rm -rf /tmp/fsm-output2
    create_dockerfile
    docker build -f docker/Dockerfile-build --target build -t fsm-build-stage . && \
    docker create --name fsm-extract fsm-build-stage && \
    docker cp fsm-extract:/go/src/factorio-server-manager/build/. /tmp/fsm-output2/ && \
    docker rm fsm-extract && \
    python3 -c "
import zipfile
z = zipfile.ZipFile('/tmp/fsm-output2/factorio-server-manager-linux.zip')
for member in z.infolist():
    parts = member.filename.split('/', 1)
    if len(parts) < 2 or not parts[1]:
        continue
    member.filename = parts[1]
    z.extract(member, '/tmp/fsm-output2/')
" && \
    cp ~/factorio-server-manager/docker/entrypoint.sh /tmp/fsm-output2/ && \
    docker build -f /tmp/Dockerfile-run -t my-fsm:latest /tmp/fsm-output2/ && \
    setup_macvlan && \
    docker run -d \
        --name ofsm \
        $(get_network_args) \
        -v ${FSM_DATA}:/opt/fsm-data \
        -v ${FACTORIO_DIR}:/opt/factorio-server \
        my-fsm:latest && \
    echo "FSM READY"
}

# Write aliases to bashrc
cat >> ~/.bashrc << ALIAS

# FSM aliases
FSM_DATA="${FSM_DATA}"
FACTORIO_DIR="${FACTORIO_DIR}"
HTTP_PORT="${HTTP_PORT}"
UDP_START="${UDP_START}"
UDP_END="${UDP_END}"

fsm-build() {
    cd ~/factorio-server-manager
    docker stop ofsm 2>/dev/null; docker rm ofsm 2>/dev/null || true
    docker builder prune -f 2>/dev/null || true
    rm -rf /tmp/fsm-output2
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
    docker build -f docker/Dockerfile-build --target build -t fsm-build-stage . && \
    docker create --name fsm-extract fsm-build-stage && \
    docker cp fsm-extract:/go/src/factorio-server-manager/build/. /tmp/fsm-output2/ && \
    docker rm fsm-extract && \
    python3 -c "
import zipfile
z = zipfile.ZipFile('/tmp/fsm-output2/factorio-server-manager-linux.zip')
for member in z.infolist():
    parts = member.filename.split('/', 1)
    if len(parts) < 2 or not parts[1]:
        continue
    member.filename = parts[1]
    z.extract(member, '/tmp/fsm-output2/')
" && \
    cp ~/factorio-server-manager/docker/entrypoint.sh /tmp/fsm-output2/ && \
    docker build -f /tmp/Dockerfile-run -t my-fsm:latest /tmp/fsm-output2/ && \
    if [ -f "\${FSM_DATA}/macvlan.conf" ]; then . \${FSM_DATA}/macvlan.conf; docker network rm fsm-lan 2>/dev/null; docker network create -d macvlan --subnet=\${SUBNET} --gateway=\${GATEWAY} -o parent=\${IFACE} fsm-lan; fi && \
    docker run -d \
        --name ofsm \
        $([ -f "\${FSM_DATA}/macvlan.conf" ] && { . \${FSM_DATA}/macvlan.conf; echo "--network fsm-lan --ip \${CONTAINER_IP}"; } || echo "-p \${HTTP_PORT}:80 -p \${UDP_START}-\${UDP_END}:\${UDP_START}-\${UDP_END}/udp") \
        -v \${FSM_DATA}:/opt/fsm-data \
        -v \${FACTORIO_DIR}:/opt/factorio-server \
        my-fsm:latest && \
    echo "FSM READY"
}

fsm-start() {
    docker stop ofsm 2>/dev/null; docker rm ofsm 2>/dev/null || true
    docker run -d \
        --name ofsm \
        $([ -f "\${FSM_DATA}/macvlan.conf" ] && { . \${FSM_DATA}/macvlan.conf; echo "--network fsm-lan --ip \${CONTAINER_IP}"; } || echo "-p \${HTTP_PORT}:80 -p \${UDP_START}-\${UDP_END}:\${UDP_START}-\${UDP_END}/udp") \
        -v \${FSM_DATA}:/opt/fsm-data \
        -v \${FACTORIO_DIR}:/opt/factorio-server \
        my-fsm:latest && echo "FSM STARTED"
}

fsm-logs() { docker logs ofsm --tail=\${1:-50} -f; }

fsm-restart() {
    docker stop ofsm 2>/dev/null; docker rm ofsm 2>/dev/null || true
    fsm-start
}

fsm-clean() {
    echo "=== FSM cleanup ==="
    for name in ofsm fsm-extract; do
        docker stop \$name 2>/dev/null && echo "stopped: \$name" || true
        docker rm \$name 2>/dev/null && echo "removed: \$name" || true
    done
    for img in my-fsm fsm-build-stage; do
        docker rmi \$img 2>/dev/null && echo "removed image: \$img" || true
    done
    rm -rf /tmp/fsm-output2 /tmp/Dockerfile-run
    read -p "Delete \${FSM_DATA} and \${FACTORIO_DIR}? [y/N]: " confirm
    if [ "\$confirm" = "y" ] || [ "\$confirm" = "Y" ]; then
        rm -rf \${FSM_DATA} \${FACTORIO_DIR}
        echo "removed data directories"
    fi
    echo "=== done ==="
}
ALIAS

echo "✓ Aliases added to ~/.bashrc"

echo ""
echo "=== Running first build ==="
do_build || { echo "Build failed! Run fsm-build manually."; exit 1; }

echo ""
echo "=== FSM Ready! ==="

# Get container IP
if [ -f "${FSM_DATA}/macvlan.conf" ]; then
    . ${FSM_DATA}/macvlan.conf
    FSM_URL="http://${CONTAINER_IP}"
else
    FSM_URL="http://$(hostname -I | awk '{print $1}'):${HTTP_PORT}"
fi

sleep 5
LOGS=$(docker logs ofsm 2>&1)
FSM_USER=$(echo "$LOGS" | grep "Username:" | tail -1 | awk '{print $NF}')
FSM_PASS=$(echo "$LOGS" | grep "admin password:" | tail -1 | awk '{print $NF}')

echo "  URL:      ${FSM_URL}"
echo "  Username: ${FSM_USER}"
echo "  Password: ${FSM_PASS}"
echo ""
echo "⚠ Change admin password after first login!"
