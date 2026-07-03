import client from "../client";

const serverPath = (serverId, suffix = "") => serverId ? `/api/servers/${serverId}${suffix}` : `/api/server${suffix}`;

export default {
    list: async () => {
        const response = await client.get('/api/servers');
        return response.data;
    },
    previewNext: async () => {
        const response = await client.get('/api/servers/next');
        return response.data;
    },
    installedVersions: async () => {
        const response = await client.get('/api/versions/installed');
        return response.data;
    },
    installVersion: async (version) => {
        const response = await client.post('/api/versions/install', {version});
        return response.data;
    },
    downloadedVersions: async () => {
        const response = await client.get('/api/versions/downloaded');
        return response.data;
    },
    deleteDownload: async (version) => {
        const response = await client.delete(`/api/versions/downloaded/${version}`);
        return response.data;
    },
    create: async (data) => {
        const response = await client.post('/api/servers', data);
        return response.data;
    },
    update: async (serverId, data) => {
        const response = await client.patch(`/api/servers/${serverId}`, data);
        return response.data;
    },
    availableVersions: async () => {
        const response = await client.get('/api/versions');
        return response.data;
    },
    factorioVersion: async (serverId) => {
        const response = await client.get(serverPath(serverId, '/facVersion'));
        return response.data;
    },
    status: async (serverId) => {
        const response = await client.get(serverId ? `/api/servers/${serverId}/status` : '/api/server/status');
        return response.data;
    },
    stop: async (serverId) => {
        const response = serverId ? await client.post(`/api/servers/${serverId}/stop`) : await client.get('/api/server/stop');
        return response.data;
    },
    start: async (ip, port, savefile, serverId) => {
        const response = await client.post(serverId ? `/api/servers/${serverId}/start` : '/api/server/start', {
            bindip: ip,
            savefile,
            port
        });
        return response.data;
    },
    installVersion: async (version) => {
        const response = await client.post('/api/server/install', {version});
        return response.data;
    },
    removeInstallation: async () => {
        const response = await client.delete('/api/server/install');
        return response.data;
    },
    save: async (serverId) => {
        const response = await client.post(`/api/servers/${serverId}/save`);
        return response.data;
    },
    kill: async (serverId) => {
        const response = serverId ? await client.post(`/api/servers/${serverId}/kill`) : await client.get('/api/server/kill');
        return response.data;
    },
    deleteServer: async (serverId) => {
        const response = await client.delete(`/api/servers/${serverId}`);
        return response.data;
    },
    installStatus: async () => {
        const response = await client.get('/api/server/install/status');
        return response.data;
    }
}
