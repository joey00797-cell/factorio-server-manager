import client from "../client";

const presets = {
    list: async (serverId) => {
        const response = await client.get(`/api/servers/${serverId}/presets`);
        return response.data;
    },
    get: async (serverId, name) => {
        const response = await client.get(`/api/servers/${serverId}/presets/${encodeURIComponent(name)}`);
        return response.data;
    },
    save: async (serverId, name, description = "") => {
        const response = await client.post(`/api/servers/${serverId}/presets`, {name, description});
        return response.data;
    },
    load: async (serverId, name) => {
        const response = await client.post(`/api/servers/${serverId}/presets/${encodeURIComponent(name)}/load`);
        return response.data;
    },
    delete: async (serverId, name) => {
        const response = await client.delete(`/api/servers/${serverId}/presets/${encodeURIComponent(name)}`);
        return response.data;
    },
    addMod: async (serverId, presetName, assetId) => {
        const response = await client.post(`/api/servers/${serverId}/presets/${encodeURIComponent(presetName)}/mods`, {asset_id: assetId});
        return response.data;
    },
    removeMod: async (serverId, presetName, modName) => {
        const response = await client.delete(`/api/servers/${serverId}/presets/${encodeURIComponent(presetName)}/mods`, {data: {mod_name: modName}});
        return response.data;
    },
};

export default presets;
