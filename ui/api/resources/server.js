import client from "../client";

export default {
    availableVersions: async () => {
        const response = await client.get('/api/server/availableVersions');
        return response.data;
    },
    factorioVersion: async () => {
        const response = await client.get('/api/server/facVersion');
        return response.data;
    },
    status: async () => {
        const response = await client.get('/api/server/status');
        return response.data;
    },
    stop: async () => {
        const response = await client.get('/api/server/stop');
        return response.data;
    },
    start: async (ip, port, savefile) => {
        const response = await client.post('/api/server/start', {
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
    kill: async () => {
        const response = await client.get('/api/server/kill');
        return response.data;
    },
    installStatus: async () => {
        const response = await client.get('/api/server/install/status');
        return response.data;
    }
}