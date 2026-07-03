import React, {useEffect, useState} from "react";
import {Navigate} from "react-router-dom";
import serverResource from "../../api/resources/server";

// Redirects unscoped routes (e.g. /mods) to a real, currently existing
// server instead of hardcoding /servers/1/... which breaks once server 1
// is deleted or was never the first one created.
const RedirectToServerScoped = ({suffix}) => {
    const [loaded, setLoaded] = useState(false);
    const [targetId, setTargetId] = useState(null);

    useEffect(() => {
        let mounted = true;
        serverResource.list()
            .then(servers => {
                if (!mounted) return;
                const sorted = (servers || []).slice().sort((a, b) => a.id - b.id);
                setTargetId(sorted.length ? sorted[0].id : null);
            })
            .catch(() => {
                if (mounted) setTargetId(null);
            })
            .finally(() => {
                if (mounted) setLoaded(true);
            });
        return () => {
            mounted = false;
        };
    }, []);

    if (!loaded) {
        return null;
    }

    if (targetId === null) {
        return <Navigate to="/" replace/>;
    }

    return <Navigate to={`/servers/${targetId}/${suffix}`} replace/>;
};

export default RedirectToServerScoped;
