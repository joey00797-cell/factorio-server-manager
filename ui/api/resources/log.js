import client from "../client";

export default {
    tail: async (serverId) => {
        const response = await client.get(serverId ? `/api/servers/${serverId}/log/tail` : '/api/log/tail');
        return response.data;
    },
    fsmTail: async () => {
        const response = await client.get('/api/fsm/log/tail');
        return response.data;
    },
}
