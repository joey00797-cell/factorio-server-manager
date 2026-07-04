import React, {useEffect, useState} from "react";
import {Navigate, Link} from "react-router-dom";
import serverResource from "../../api/resources/server";
import {useTranslation} from "react-i18next";

// Redirects unscoped routes (e.g. /mods) to a real, currently existing
// server instead of hardcoding /servers/1/... which breaks once server 1
// is deleted or was never the first one created.
const RedirectToServerScoped = ({suffix}) => {
    const {t} = useTranslation();
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
        return (
            <div className="flex flex-col items-center justify-center p-12 text-center">
                <div className="bg-gray-dark accentuated p-8 max-w-md">
                    <h2 className="text-dirty-white text-2xl font-bold mb-3">{t("servers.no_servers_title", "No servers yet")}</h2>
                    <p className="text-gray-400 mb-6">{t("servers.no_servers_hint", "Create your first Factorio server to get started.")}</p>
                    <Link
                        to="/"
                        className="bg-green py-2 px-6 hover:glow-orange hover:bg-orange accentuated text-black font-bold inline-block"
                    >
                        {t("servers.go_create", "Create server")}
                    </Link>
                </div>
            </div>
        );
    }

    return <Navigate to={`/servers/${targetId}/${suffix}`} replace/>;
};

export default RedirectToServerScoped;
