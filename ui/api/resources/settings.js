import client from "../client";

const serverPrefix = serverId => serverId ? `/api/servers/${serverId}` : '/api';

export default {
    server: {
        list: async (serverId) => {
            const response = await client.get(`${serverPrefix(serverId)}/settings`)
            return response.data;
        },
        update: async (data, serverId) => {
            const response = await client.post(`${serverPrefix(serverId)}/settings/update`, data)
            return response.data;
        }
    },
    game: {
        list: async (serverId) => {
            const response = await client.get(`${serverPrefix(serverId)}/config`);
            return response.data;
        },
        update: async (data, serverId) => {
            const response = await client.post(`${serverPrefix(serverId)}/config/update`, data);
            return response.data;
        }
    }
}
