import React, {useCallback, useEffect, useMemo, useState} from "react";
import {Link} from "react-router-dom";
import Panel from "../components/Panel";
import Button from "../components/Button";
import serverResource from "../../api/resources/server";
import savesResource from "../../api/resources/saves";
import {useTranslation} from "react-i18next";
import {useServers} from "../context/ServersContext";

const emptyCreate = {
    name: "",
    version: "stable",
    bind_ip: "0.0.0.0",
    port: "",
    autostart: false,
};

const SAVE_REFRESH_DELAYS_MS = [1200, 3500];
const STOP_RECONCILE_TIMEOUT_MS = 20000;

const Controls = () => {
    const { t } = useTranslation();
    const {servers, refreshServers, refreshServerUntil} = useServers();
    const [availableVersions, setAvailableVersions] = useState({});
    const [createForm, setCreateForm] = useState(emptyCreate);
    const [savesByServer, setSavesByServer] = useState({});
    const [selectedSaveByServer, setSelectedSaveByServer] = useState({});
    const [busy, setBusy] = useState({});

    const serverIdsKey = useMemo(() => servers.map(server => server.id).join(","), [servers]);

    const loadSaves = useCallback(async (srv) => {
        const serverId = typeof srv === "object" ? srv?.id : srv;
        if (!serverId) return [];

        const saves = await savesResource.list(true, serverId);
        setSavesByServer(prev => ({...prev, [serverId]: saves || []}));
        setSelectedSaveByServer(prev => {
            const names = new Set((saves || []).map(save => save.name));
            if (prev[serverId] && names.has(prev[serverId])) return prev;
            const latest = (saves || []).find(save => save.name.startsWith("Load Latest"));
            return {...prev, [serverId]: latest?.name || saves?.[0]?.name || ""};
        });
        return saves || [];
    }, []);

    useEffect(() => {
        refreshServers().catch(err => console.error("Error loading servers", err));
        serverResource.availableVersions().then(setAvailableVersions).catch(() => {});
    }, [refreshServers]);

    useEffect(() => {
        servers.forEach(server => {
            loadSaves(server).catch(err => console.error("Error loading saves", err));
        });
    }, [serverIdsKey, loadSaves]);

    const versionLabel = (type) => {
        const v = availableVersions?.[type]?.headless;
        return v ? `${type} (${v})` : type;
    };

    const setBusyFor = (id, action, value) => {
        setBusy(prev => ({...prev, [`${id}:${action}`]: value}));
    };

    const scheduleSaveRefresh = useCallback(srv => {
        SAVE_REFRESH_DELAYS_MS.forEach(delay => {
            setTimeout(() => {
                loadSaves(srv).catch(err => console.error("Error refreshing saves", err));
            }, delay);
        });
    }, [loadSaves]);

    const reconcileAfterAction = async (srv, action) => {
        if (action === "save") {
            await loadSaves(srv);
            scheduleSaveRefresh(srv);
            return;
        }

        if (action === "stop" || action === "kill") {
            await refreshServerUntil(
                srv.id,
                server => !!server && !server.running,
                {timeoutMs: STOP_RECONCILE_TIMEOUT_MS}
            );
            await refreshServers();
            await loadSaves(srv);
            return;
        }

        await refreshServers();

        if (action === "start") {
            await loadSaves(srv);
        }
    };

    const runAction = async (srv, action, fn) => {
        setBusyFor(srv.id, action, true);
        try {
            await fn();
            await reconcileAfterAction(srv, action);
        } finally {
            setBusyFor(srv.id, action, false);
        }
    };

    const createServer = async (e) => {
        e.preventDefault();
        const payload = {
            ...createForm,
            port: createForm.port ? parseInt(createForm.port) : 0,
        };
        await serverResource.create(payload);
        setCreateForm(emptyCreate);
        await refreshServers();
    };

    return (
        <>
            <Panel
                className="mb-6"
                title={t("servers.create", "Create server")}
                content={
                    <form className="grid gap-3 lg:grid-cols-5" onSubmit={createServer}>
                        <input
                            className="shadow appearance-none border py-2 px-3 text-black"
                            placeholder={t("name")}
                            value={createForm.name}
                            onChange={e => setCreateForm({...createForm, name: e.target.value})}
                        />
                        <select
                            className="shadow appearance-none border py-2 px-3 text-black"
                            value={createForm.version}
                            onChange={e => setCreateForm({...createForm, version: e.target.value})}
                        >
                            <option value="stable">{versionLabel("stable")}</option>
                            <option value="experimental">{versionLabel("experimental")}</option>
                        </select>
                        <input
                            className="shadow appearance-none border py-2 px-3 text-black"
                            value={createForm.bind_ip}
                            onChange={e => setCreateForm({...createForm, bind_ip: e.target.value})}
                        />
                        <input
                            className="shadow appearance-none border py-2 px-3 text-black"
                            placeholder={t("controls.port")}
                            type="number"
                            min={1}
                            max={65535}
                            value={createForm.port}
                            onChange={e => setCreateForm({...createForm, port: e.target.value})}
                        />
                        <Button isSubmit={true} type="success" className="w-full">{t("create")}</Button>
                    </form>
                }
            />

            <div className="grid gap-4 xl:grid-cols-2">
                {servers.map(srv => (
                    <ServerCard
                        key={srv.id}
                        server={srv}
                        saves={savesByServer[srv.id] || []}
                        selectedSave={selectedSaveByServer[srv.id] || ""}
                        setSelectedSave={save => setSelectedSaveByServer(prev => ({...prev, [srv.id]: save}))}
                        busy={busy}
                        runAction={runAction}
                        t={t}
                    />
                ))}
            </div>
        </>
    );
};

const ServerCard = ({server, saves, selectedSave, setSelectedSave, busy, runAction, t}) => {
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
    const versionText = useMemo(() => {
        if (server.fac_version && server.fac_version !== "0.0.0.0") return server.fac_version;
        return server.version || t("controls.unknown");
    }, [server, t]);

    const running = !!server.running;
    const noSave = saves.length === 0;
    const deleteDisabledReason = running ? t("servers.cant_delete_running", "Stop this server before deleting it.") : "";
    const startDisabledReason = noSave ? t("servers.cant_start_no_save", "Create or upload a save before starting this server.") : "";

    useEffect(() => {
        if (!isConfirmingDelete) return;
        const timeout = setTimeout(() => setIsConfirmingDelete(false), 8000);
        return () => clearTimeout(timeout);
    }, [isConfirmingDelete]);

    const deleteServer = () => {
        if (!isConfirmingDelete) {
            setIsConfirmingDelete(true);
            return;
        }
        runAction(server, "delete", () => serverResource.deleteServer(server.id))
            .finally(() => setIsConfirmingDelete(false));
    };

    return (
        <div className="bg-gray-dark accentuated p-4">
            <div className="flex items-start justify-between gap-3 mb-4">
                <div>
                    <h2 className="text-dirty-white text-xl font-bold">{server.name || `Server ${server.id}`}</h2>
                    <div className={running ? "text-green font-bold" : "text-red font-bold"}>
                        {running ? t("controls.running") : t("controls.stopped")}
                        {server.pending_restart ? <span className="ml-2 text-orange">({t("servers.pending_restart", "restart pending")})</span> : null}
                    </div>
                </div>
                <Link
                    className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold"
                    to={`/servers/${server.id}/saves`}
                >
                    {t("servers.manage", "Manage")}
                </Link>
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
                <Info label="IP" value={server.bindip || server.bind_ip || "0.0.0.0"}/>
                <Info label={t("controls.port")} value={server.port || "-"}/>
                <Info label={t("controls.f_version")} value={versionText}/>
                <Info label={t("controls.save")} value={server.savefile || "-"}/>
            </div>

            {!running && (
                <div className="mb-3">
                    <select
                        className="shadow appearance-none border w-full py-2 px-3 text-black"
                        value={selectedSave}
                        disabled={noSave}
                        onChange={e => setSelectedSave(e.target.value)}
                    >
                        {noSave
                            ? <option value="">{t("saves.empty", "No saves for this server yet.")}</option>
                            : saves.map(save => <option key={save.name} value={save.name}>{save.name}</option>)}
                    </select>
                </div>
            )}

            <div className="flex flex-wrap gap-2">
                {running ? (
                    <>
                        <Button
                            size="sm"
                            type="default"
                            isLoading={busy[`${server.id}:save`]}
                            onClick={() => runAction(server, "save", () => serverResource.save(server.id))}
                        >
                            {t("servers.save_now", "Save")}
                        </Button>
                        <Button
                            size="sm"
                            type="default"
                            isLoading={busy[`${server.id}:stop`]}
                            onClick={() => runAction(server, "stop", () => serverResource.stop(server.id))}
                        >
                            {t("controls.save&stop")}
                        </Button>
                        <Button
                            size="sm"
                            type="danger"
                            isLoading={busy[`${server.id}:kill`]}
                            onClick={() => runAction(server, "kill", () => serverResource.kill(server.id))}
                        >
                            {t("controls.kill_server")}
                        </Button>
                    </>
                ) : (
                    <span className={`inline-block ${startDisabledReason ? "cursor-not-allowed" : ""}`} title={startDisabledReason || undefined}>
                        <Button
                            className={startDisabledReason ? "pointer-events-none" : ""}
                            size="sm"
                            type="success"
                            isDisabled={!!startDisabledReason}
                            isLoading={busy[`${server.id}:start`]}
                            onClick={() => runAction(server, "start", () => serverResource.start(server.bindip || server.bind_ip || "0.0.0.0", server.port || 34197, selectedSave, server.id))}
                        >
                            {t("controls.start_server")}
                        </Button>
                    </span>
                )}
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold" to={`/servers/${server.id}/mods`}>
                    {t("mods.title")}
                </Link>
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold" to={`/servers/${server.id}/server-settings`}>
                    {t("server_settings.title")}
                </Link>
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold" to={`/servers/${server.id}/mod-options`}>
                    {t("mods.mod_options", "Mod Options")}
                </Link>
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold" to={`/servers/${server.id}/console`}>
                    {t("console.title")}
                </Link>
                <span className={`inline-block ${deleteDisabledReason ? "cursor-not-allowed" : ""}`} title={deleteDisabledReason || undefined}>
                    <Button
                        className={deleteDisabledReason ? "pointer-events-none" : ""}
                        size="sm"
                        type="danger"
                        isDisabled={!!deleteDisabledReason}
                        isLoading={busy[`${server.id}:delete`]}
                        onClick={deleteServer}
                    >
                        {isConfirmingDelete ? t("servers.confirm_delete", "Confirm delete") : t("servers.delete", "Delete Server")}
                    </Button>
                </span>
            </div>
        </div>
    );
};

const Info = ({label, value}) => (
    <div>
        <div className="font-bold">{label}</div>
        <div className="break-words">{value}</div>
    </div>
);

export default Controls;
