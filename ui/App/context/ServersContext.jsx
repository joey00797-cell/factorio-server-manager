import React, {createContext, useCallback, useContext, useEffect, useMemo, useState} from "react";
import serverApi from "../../api/resources/server";
import socket from "../../api/socket";

const SERVER_REFRESH_INTERVAL_MS = 10000;
const SERVER_POLL_INTERVAL_MS = 750;
const SERVER_POLL_TIMEOUT_MS = 20000;

const ServersContext = createContext(null);

const wait = ms => new Promise(resolve => setTimeout(resolve, ms));

const parseServerUpdate = message => {
    if (!message) return null;
    return typeof message === "string" ? JSON.parse(message) : message;
};

const mergeServer = (servers, updatedServer) => {
    if (!updatedServer?.id) return servers;

    const index = servers.findIndex(server => server.id === updatedServer.id);
    if (index === -1) {
        return [...servers, updatedServer];
    }

    return servers.map(server => server.id === updatedServer.id ? {...server, ...updatedServer} : server);
};

export const ServersProvider = ({children}) => {
    const [servers, setServers] = useState([]);

    const mergeServerUpdate = useCallback(updatedServer => {
        setServers(previous => mergeServer(previous, updatedServer));
    }, []);

    const refreshServers = useCallback(async () => {
        const response = await serverApi.list();
        const nextServers = response || [];
        setServers(nextServers);
        return nextServers;
    }, []);

    const refreshServer = useCallback(async serverId => {
        if (!serverId) {
            const refreshed = await refreshServers();
            return refreshed?.[0] || null;
        }

        const server = await serverApi.status(serverId);
        if (server?.id) {
            mergeServerUpdate(server);
        } else {
            await refreshServers();
        }
        return server;
    }, [mergeServerUpdate, refreshServers]);

    const refreshServerUntil = useCallback(async (serverId, predicate, options = {}) => {
        const timeoutMs = options.timeoutMs ?? SERVER_POLL_TIMEOUT_MS;
        const intervalMs = options.intervalMs ?? SERVER_POLL_INTERVAL_MS;
        const startedAt = Date.now();
        let latest = await refreshServer(serverId);

        if (predicate(latest)) {
            return latest;
        }

        while (Date.now() - startedAt < timeoutMs) {
            await wait(intervalMs);
            latest = await refreshServer(serverId);
            if (predicate(latest)) {
                return latest;
            }
        }

        return latest;
    }, [refreshServer]);

    useEffect(() => {
        let cancelled = false;

        const safeRefreshServers = async () => {
            try {
                const response = await serverApi.list();
                if (!cancelled) {
                    setServers(response || []);
                }
            } catch (err) {
                if (!cancelled) {
                    console.error("Error loading server summary", err);
                }
            }
        };

        const handleServerUpdate = message => {
            try {
                const updatedServer = parseServerUpdate(message);
                if (!updatedServer?.id) {
                    safeRefreshServers();
                    return;
                }
                mergeServerUpdate(updatedServer);
            } catch (err) {
                console.error("Error handling server status update", err);
            }
        };

        safeRefreshServers();
        socket.emit("servers subscribe");
        socket.on("servers", handleServerUpdate);
        const interval = setInterval(safeRefreshServers, SERVER_REFRESH_INTERVAL_MS);

        return () => {
            cancelled = true;
            clearInterval(interval);
            socket.off("servers", handleServerUpdate);
            socket.emit("servers unsubscribe");
        };
    }, [mergeServerUpdate]);

    const value = useMemo(() => ({
        servers,
        refreshServers,
        refreshServer,
        refreshServerUntil,
        mergeServerUpdate,
    }), [servers, refreshServers, refreshServer, refreshServerUntil, mergeServerUpdate]);

    return (
        <ServersContext.Provider value={value}>
            {children}
        </ServersContext.Provider>
    );
};

export const useServers = () => {
    const context = useContext(ServersContext);
    if (!context) {
        throw new Error("useServers must be used within ServersProvider");
    }
    return context;
};
