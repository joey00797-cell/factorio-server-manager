import client from "../client";

const modLibrary = {
    // Библиотека
    list: async () => {
        const response = await client.get("/api/mods/library");
        return response.data;
    },

    upload: async (file) => {
        const formData = new FormData();
        formData.append("mod", file);
        const response = await client.post("/api/mods/library/upload", formData, {
            headers: {"Content-Type": "multipart/form-data"},
        });
        return response.data;
    },

    portalImport: async (downloadURL, fileName, modName) => {
        const response = await client.post("/api/mods/library/portal-import", {
            download_url: downloadURL,
            file_name: fileName,
            mod_name: modName,
        });
        return response.data;
    },

    delete: async (assetId) => {
        const response = await client.delete(`/api/mods/library/${assetId}`);
        return response.data;
    },

    // Манифест сервера
    manifest: {
        get: async (serverId) => {
            const response = await client.get(`/api/servers/${serverId}/mods/manifest`);
            return response.data;
        },

        update: async (serverId, items) => {
            const response = await client.put(`/api/servers/${serverId}/mods/manifest`, {items});
            return response.data;
        },

        preview: async (serverId) => {
            const response = await client.get(`/api/servers/${serverId}/mods/manifest/preview`);
            return response.data;
        },

        apply: async (serverId) => {
            const response = await client.post(`/api/servers/${serverId}/mods/manifest/apply`);
            return response.data;
        },
    },
};

export default modLibrary;
