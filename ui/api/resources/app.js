import client from "../client";

const app = {
    version: async () => {
        const response = await client.get("/api/app/version");
        return response.data;
    }
};

export default app;
