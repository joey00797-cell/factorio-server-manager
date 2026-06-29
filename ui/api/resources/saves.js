import client from "../client";

const base = serverId => serverId ? `/api/servers/${serverId}/saves` : '/api/saves';

export default {
    list: async (latest, serverId) => {
        const response = await client.get(`${base(serverId)}/list`, {
            params: {
                latest
            }
        });
        return response.data;
    },
    delete: async (save, serverId) => {
        const response = await client.get(`${base(serverId)}/rm/${save.name}`);
        return response.data;
    },
    create: async (name, serverId) => {
        const response = await client.get(`${base(serverId)}/create/${name}`);
        return response.data;
    },
    upload: async (file, serverId) => {
        let formData = new FormData();
        formData.append("savefile", file);

        const response = await client.post(`${base(serverId)}/upload`, formData, {
            headers: {
                "Content-Type": "multipart/form-data"
            }
        });
        return response.data;
    },
    downloadURL: (saveName, serverId) => `${base(serverId)}/dl/${saveName}`,
    mods: async (save, serverId) => {
        const response = await client.post(serverId ? `/api/servers/${serverId}/saves/mods` : "/api/saves/mods", {
            saveFile: save
        });
        return response.data;
    }
}
