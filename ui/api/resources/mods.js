import client from "../client";

const base = serverId => serverId ? `/api/servers/${serverId}/mods` : '/api/mods';
const savesBase = serverId => serverId ? `/api/servers/${serverId}/saves` : '/api/saves';

const mods = {
    installed: async (serverId) => {
        const response = await client.get(`${base(serverId)}/list`);
        return response.data;
    },
    toggle: async (name, serverId) => {
        const response = await client.post(`${base(serverId)}/toggle`, {name});
        return response.data;
    },
    delete: async (name, serverId) => {
        const response = await client.post(`${base(serverId)}/delete`, {name});
        return response.data;
    },
    update: async ({modName, downloadUrl, fileName}, serverId) => {
        const response = await client.post(`${base(serverId)}/update`, {modName, downloadUrl, fileName})
        return response.data;
    },
    upload: async (file, serverId) => {
        let formData = new FormData();
        formData.append("mod_file", file);

        const response = await client.post(`${base(serverId)}/upload`, formData, {
            headers: {
                "Content-Type": "multipart/form-data"
            }
        });
        return response.data;
    },
    deleteAll: async (serverId) => {
        const response = await client.post(`${base(serverId)}/delete/all`);
        return response.data;
    },
    getFromSave: async (saveFile, serverId) => {
        const response = await client.post(`${savesBase(serverId)}/mods/list`, {saveFile});
        return response.data;
    },
    syncFromSave: async (saveFile, modNames, serverId) => {
        const response = await client.post(`${savesBase(serverId)}/mods/sync`, {saveFile, modNames});
        return response.data;
    },
    downloadAllURL: serverId => `${base(serverId)}/download`,
    modSettings: {
        get: async (serverId) => {
            const response = await client.get(`/api/servers/${serverId}/mod-settings`);
            return response.data;
        },
        update: async (serverId, data) => {
            const response = await client.post(`/api/servers/${serverId}/mod-settings`, data);
            return response.data;
        }
    },
    portal: {
        login: async (username, token) => {
            const response = await client.post('/api/mods/portal/login', {
                username,
                token
            });
            return response.data;
        },
        status: async () => {
            const response = await client.get('/api/mods/portal/loginstatus');
            return response.data;
        },
        logout: async () => {
            const response = await client.get('/api/mods/portal/logout');
            return response.data
        },
        installMultiple: async (mods, serverId) => {
            const response = await client.post(serverId ? `/api/servers/${serverId}/mods/portal/install/multiple` : '/api/mods/portal/install/multiple', mods);
            return response.data
        },
        install: async (downloadUrl, fileName, modName, serverId) => {
            const response = await client.post(serverId ? `/api/servers/${serverId}/mods/portal/install` : '/api/mods/portal/install', {
                downloadUrl,
                fileName,
                modName
            });
            return response.data
        },
        list: async () => {
            const response = await client.get('/api/mods/portal/list');
            return response.data
        },
        info: async mod => {
            const response = await client.get(`/api/mods/portal/info/${mod}`);
            return response.data;
        }
    },
    packs: {
        list: async () => {
            const response = await client.get('/api/mods/packs/list');
            return response.data;
        },
        create: async name => {
            const response = await client.post('/api/mods/packs/create', {name});
            return response.data;
        },
        delete: async name => {
            const response = await client.post(`/api/mods/packs/${name}/delete`);
            return response.data;
        },
        download: async name => {
            const response = await client.get(`/api/mods/packs/${name}/download`);
            return response.data;
        },
        load: async name => {
            const response = await client.post(`/api/mods/packs/${name}/load`);
            return response.data;
        },
        mods: {
            list: async packName => {
                const response = await client.get(`/api/mods/packs/${packName}/list`);
                return response.data;
            },
            toggle: async (packName, modName) => {
                const response = await client.post(`/api/mods/packs/${packName}/mod/toggle`, {
                    name: modName
                });
                return response.data;
            },
            update: async (packName, {modName, downloadUrl, fileName}) => {
                const response = await client.post(`/api/mods/packs/${packName}/mod/update`, {modName, downloadUrl, fileName})
                return response.data;
            },
            delete: async (packName, modName) => {
                const response = await client.post(`/api/mods/packs/${packName}/mod/delete`, {
                    name: modName
                });
                return response.data;
            },
        }
    }
,
    cancelSync: async () => {
        const response = await client.post('/api/saves/mods/sync/cancel');
        return response.data;
    }
}

export default mods;
