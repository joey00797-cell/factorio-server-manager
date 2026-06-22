export default {
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
    kill: async () => {
        const response = await client.get('/api/server/kill');
        return response.data;
    },
    version: {
        current: async () => {
            const response = await client.get('/api/server/version/current');
            return response.data;
        },
        available: async () => {
            const response = await client.get('/api/server/version/available');
            return response.data;
        },
        list: async () => {
            const response = await client.get('/api/server/version/list');
            return response.data;
        },
        install: async (version) => {
            const response = await client.post('/api/server/version/install', { version });
            return response.data;
        },
        installStatus: async () => {
            const response = await client.get('/api/server/version/install-status');
            return response.data;
        }
    }
};