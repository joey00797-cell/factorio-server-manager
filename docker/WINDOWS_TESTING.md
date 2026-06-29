# Windows Docker Testing

Use this flow with Docker Desktop on Windows using Linux containers.

## Start

From the repository root:

```powershell
.\scripts\docker-windows-test.ps1 start
```

The script builds the Linux release artifact, builds a local runtime image, starts Docker Compose, and prints the UI URL:

```text
http://localhost:8080
```

The compose file uses named volumes for `/opt/fsm-data` and `/opt/factorio-server` so Windows bind-mount permissions do not get in the way.

## Logs

```powershell
.\scripts\docker-windows-test.ps1 logs
```

## Stop

```powershell
.\scripts\docker-windows-test.ps1 stop
```

## Reset

This removes the containers and named volumes, including managed servers and downloaded Factorio versions:

```powershell
.\scripts\docker-windows-test.ps1 reset
```

## Manual Acceptance

1. Open `http://localhost:8080`.
2. Confirm the migrated/default server appears as server `1`.
3. Install or select a Factorio version from the UI flow.
4. Create server `2`.
5. Create or upload saves for each server.
6. Start both servers on different UDP ports in the mapped `34197-34205` range.
7. Use the server card `Save` action and confirm the server keeps running.
8. Stop one server and confirm the other remains running.
