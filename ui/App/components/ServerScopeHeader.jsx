import React, {useEffect, useState} from "react";
import {useParams} from "react-router-dom";
import {useTranslation} from "react-i18next";
import serverResource from "../../api/resources/server";

const ServerScopeHeader = () => {
    const {serverId} = useParams();
    const {t} = useTranslation();
    const [server, setServer] = useState(null);

    useEffect(() => {
        if (!serverId) return;
        let mounted = true;
        serverResource.status(serverId)
            .then(data => {
                if (mounted) setServer(data);
            })
            .catch(() => {
                if (mounted) setServer(null);
            });
        return () => {
            mounted = false;
        };
    }, [serverId]);

    if (!serverId) return null;

    const name = server?.name || `${t("servers.server", "Server")} ${serverId}`;
    const running = !!server?.running;
    const version = server?.fac_version && server.fac_version !== "0.0.0.0"
        ? server.fac_version
        : server?.version || t("controls.unknown");
    const bindIP = server?.bindip || server?.bind_ip || "0.0.0.0";
    const port = server?.port || "-";
    const save = server?.savefile || "-";

    return (
        <div className="mb-6 bg-black accentuated p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <div className="text-sm font-bold text-orange uppercase">{t("servers.editing", "Editing server")}</div>
                    <h1 className="text-dirty-white text-2xl font-bold">{name}</h1>
                </div>
                <div className="flex flex-wrap gap-2 text-sm">
                    <Badge label={t("servers.id", "ID")} value={serverId}/>
                    <Badge
                        label={t("controls.status")}
                        value={running ? t("controls.running") : t("controls.stopped")}
                        tone={running ? "green" : "red"}
                    />
                    <Badge label="IP" value={bindIP}/>
                    <Badge label={t("controls.port")} value={port}/>
                    <Badge label={t("controls.f_version")} value={version}/>
                    <Badge label={t("controls.save")} value={save}/>
                </div>
            </div>
        </div>
    );
};

const Badge = ({label, value, tone}) => {
    const toneClass = tone === "green" ? "text-green" : tone === "red" ? "text-red" : "text-dirty-white";
    return (
        <div className="bg-gray-dark px-3 py-2 accentuated">
            <span className="font-bold">{label}: </span>
            <span className={toneClass}>{value}</span>
        </div>
    );
};

export default ServerScopeHeader;
