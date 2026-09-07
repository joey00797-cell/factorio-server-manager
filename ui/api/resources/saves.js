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
    },
    backups: {
        list: async (serverId) => {
            const response = await client.get(`/api/servers/${serverId}/saves/backups`);
            return response.data;
        },
        listAll: async (serverId) => {
            const response = await client.get(`/api/servers/${serverId}/saves/backups/all`);
            return response.data;
        },
        schedule: async (serverId) => {
            const response = await client.get(`/api/servers/${serverId}/saves/backups/schedule`);
            return response.data;
        },
        updateSchedule: async (serverId, schedule) => {
            const response = await client.put(`/api/servers/${serverId}/saves/backups/schedule`, schedule);
            return response.data;
        },
        run: async (serverId) => {
            const response = await client.post(`/api/servers/${serverId}/saves/backups/run`);
            return response.data;
        },
        remove: async (serverId, name) => {
            const response = await client.delete(`/api/servers/${serverId}/saves/backups/${name}`);
            return response.data;
        },
        downloadUrl: (serverId, name) => `/api/servers/${serverId}/saves/backups/${name}/download`,
        restore: async (serverId, name, targetServerId, sourceServerId) => {
            const response = await client.post(`/api/servers/${serverId}/saves/backups/${name}/restore`, {target_server_id: targetServerId || serverId, source_server_id: sourceServerId || serverId});
            return response.data;
        },
        pin: async (serverId, name) => {
            const response = await client.post(`/api/servers/${serverId}/saves/backups/${name}/pin`);
            return response.data;
        },
        rename: async (serverId, name, newName) => {
            const response = await client.put(`/api/servers/${serverId}/saves/backups/${name}/rename`, {new_name: newName});
            return response.data;
        },
    },
}
