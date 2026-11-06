#!/bin/bash
# FSM native installer / updater (Linux, no Docker)
# Env: FSM_DIR (install dir), FSM_YES=1 (no questions), FSM_VERSION=vX.Y.Z (pin a release),
#      FSM_INSECURE=1 (accept a release without SHA256SUMS)
REPO="joey00797-cell/factorio-server-manager"

main() {
    set -u
    have() { [ -n "$(command -v "$1")" ]; }
    interactive() { [ "${FSM_YES:-0}" != "1" ] && { true < /dev/tty; } 2>/dev/null; }
    ask() { # ask "question [Y/n]" default(y/n)
        local a=""
        if ! interactive; then [ "$2" = "y" ]; return; fi
        read -r -p "$1 " a < /dev/tty
        case "$a" in y|Y) return 0;; n|N) return 1;; "") [ "$2" = "y" ];; *) return 1;; esac
    }
    sha256_of() {
        if have sha256sum; then sha256sum "$1" | awk '{print $1}'
        elif have shasum; then shasum -a 256 "$1" | awk '{print $1}'
        else echo ""; fi
    }

    have curl || { echo "Missing: curl"; exit 1; }
    have unzip || have python3 || { echo "Need unzip or python3"; exit 1; }

    echo "Checking the release..."
    local api="releases/latest"
    [ -n "${FSM_VERSION:-}" ] && api="releases/tags/$FSM_VERSION"
    local info tag url sums_url
    info=$(curl -fsSL "https://api.github.com/repos/$REPO/$api") || { echo "Cannot reach GitHub or release not found"; exit 1; }
    tag=$(echo "$info" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
    url=$(echo "$info" | sed -n 's/.*"browser_download_url": *"\([^"]*linux[^"]*\.zip\)".*/\1/p' | head -1)
    sums_url=$(echo "$info" | sed -n 's/.*"browser_download_url": *"\([^"]*SHA256SUMS\)".*/\1/p' | head -1)
    [ -n "$tag" ] && [ -n "$url" ] || { echo "No Linux archive in the release ($tag)."; exit 1; }

    local def="/opt/fsm" dir a=""
    [ "$(id -u)" != "0" ] && def="$HOME/fsm"
    dir="${FSM_DIR:-$def}"
    if [ -z "${FSM_DIR:-}" ] && interactive; then
        read -r -p "Install directory [$dir]: " a < /dev/tty
        dir="${a:-$dir}"
    fi

    local cur="" update=0
    [ -f "$dir/.fsm-version" ] && cur=$(cat "$dir/.fsm-version")
    if [ -f "$dir/factorio-server-manager" ]; then
        if [ "$cur" = "$tag" ]; then
            echo "FSM $tag is already installed in $dir."; exit 0
        fi
        echo "Installed: ${cur:-unknown}, available: $tag"
        ask "Update now? [Y/n]" y || { echo "Cancelled."; exit 0; }
        update=1
    else
        echo "Release: $tag, will install into $dir"
        ask "Continue? [Y/n]" y || { echo "Cancelled."; exit 0; }
    fi

    # a running binary cannot be replaced
    if [ "$update" = "1" ]; then
        local p
        for p in /proc/[0-9]*; do
            if [ "$(readlink "$p/exe")" = "$dir/factorio-server-manager" ]; then
                echo "FSM is running from $dir (pid ${p#/proc/}). Stop it and run this again."; exit 1
            fi
        done
    fi

    tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
    echo "Downloading $url"
    curl -fL --progress-bar -o "$tmp/fsm.zip" "$url" || { echo "Download failed"; exit 1; }

    # integrity check
    if [ -n "$sums_url" ]; then
        curl -fsSL -o "$tmp/SHA256SUMS" "$sums_url" || { echo "Cannot download SHA256SUMS"; exit 1; }
        local name="${url##*/}" want got
        want=$(awk -v n="$name" '$2 == n || $2 == "*" n {print $1; exit}' "$tmp/SHA256SUMS")
        got=$(sha256_of "$tmp/fsm.zip")
        [ -n "$want" ] || { echo "$name is not listed in SHA256SUMS"; exit 1; }
        [ -n "$got" ] || { echo "No sha256sum/shasum available to verify the download"; exit 1; }
        [ "$want" = "$got" ] || { echo "Checksum mismatch for $name (expected $want, got $got). Aborting."; exit 1; }
        echo "Checksum OK"
    else
        echo "WARNING: this release has no SHA256SUMS, the download cannot be verified."
        if [ "${FSM_INSECURE:-0}" != "1" ]; then
            ask "Continue without verification? [y/N]" n || { echo "Cancelled."; exit 1; }
        fi
    fi

    mkdir -p "$tmp/x"
    if have unzip; then
        unzip -q "$tmp/fsm.zip" -d "$tmp/x" || exit 1
    else
        python3 -c "import zipfile,sys; zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2])" "$tmp/fsm.zip" "$tmp/x" || exit 1
    fi
    local src="$tmp/x/factorio-server-manager"
    [ -f "$src/factorio-server-manager" ] || { echo "Unexpected archive layout"; exit 1; }

    mkdir -p "$dir" || exit 1
    if [ "$update" = "1" ]; then
        cp -a "$dir/factorio-server-manager" "$dir/factorio-server-manager.bak"
        rm -rf "$dir/app"
        cp -a "$src/app" "$dir/app"
        cp -a "$src/factorio-server-manager" "$src/start.sh" "$dir/"
    else
        cp -a "$src/." "$dir/"
    fi
    chmod +x "$dir/factorio-server-manager" "$dir/start.sh"
    echo "$tag" > "$dir/.fsm-version"

    echo ""
    echo "FSM $tag is ready in $dir"
    echo "Start:  cd $dir && ./start.sh"
}

main "$@"
