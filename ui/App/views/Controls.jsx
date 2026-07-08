import React, {useCallback, useEffect, useMemo, useState} from "react";
import {Link} from "react-router-dom";
import Panel from "../components/Panel";
import Button from "../components/Button";
import serverResource from "../../api/resources/server";
import savesResource from "../../api/resources/saves";
import {useTranslation} from "react-i18next";
import {useServers} from "../context/ServersContext";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faTrash} from "@fortawesome/free-solid-svg-icons";

const emptyCreate = {
    name: "",
    version: "stable",
    bind_ip: "0.0.0.0",
    port: "",
    autostart: false,
};

const SAVE_REFRESH_DELAYS_MS = [1200, 3500];
const STOP_RECONCILE_TIMEOUT_MS = 20000;
const START_RECONCILE_TIMEOUT_MS = 120000;

const Controls = () => {
    const { t } = useTranslation();
    const {servers, refreshServers, refreshServerUntil} = useServers();
    const [availableVersions, setAvailableVersions] = useState({});
    const [installedVersions, setInstalledVersions] = useState([]);
    const [downloadedVersions, setDownloadedVersions] = useState([]);
    const [busyVersion, setBusyVersion] = useState({});
    const [expandedDelete, setExpandedDelete] = useState(false);
    const [confirmDeleteAll, setConfirmDeleteAll] = useState(false);
    const [createForm, setCreateForm] = useState(emptyCreate);
    const [nextPreview, setNextPreview] = useState({name: "", port: ""});
    const [savesByServer, setSavesByServer] = useState({});
    const [selectedSaveByServer, setSelectedSaveByServer] = useState({});
    const [busy, setBusy] = useState({});

    const serverIdsKey = useMemo(() => servers.map(server => server.id).join(","), [servers]);

    const fetchNextPreview = async () => {
        try {
            const preview = await serverResource.previewNext();
            setNextPreview({name: preview?.name || "", port: preview?.port || ""});
        } catch {
            setNextPreview({name: "", port: ""});
        }
    };

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
        fetchNextPreview();
        serverResource.availableVersions().then(setAvailableVersions).catch(() => {});
        serverResource.installedVersions().then(d => setInstalledVersions(d?.versions || [])).catch(() => {});
        serverResource.downloadedVersions().then(d => setDownloadedVersions(d?.versions || [])).catch(() => {});
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
            window.flash(srv.name + ": " + (srv.savefile || "saved"), "green");
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

        if (action === "start") {
            await refreshServerUntil(
                srv.id,
                server => !!server && !!server.rcon_connected,
                {timeoutMs: START_RECONCILE_TIMEOUT_MS}
            );
            await refreshServers();
            await loadSaves(srv);
            return;
        }

        await refreshServers();
    };

    const runAction = async (srv, action, fn) => {
        setBusyFor(srv.id, action, true);
        try {
            await fn();
            await reconcileAfterAction(srv, action);
            const messages = {
                start: {message: srv.name + " started", color: "green"},
                stop:  {message: srv.name + " stopped", color: "orange"},
                kill:  {message: srv.name + " killed",  color: "red"},
            };
            if (messages[action]) window.flash(messages[action].message, messages[action].color);
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
        await fetchNextPreview();
        serverResource.installedVersions().then(d => setInstalledVersions(d?.versions || [])).catch(() => {});
        serverResource.downloadedVersions().then(d => setDownloadedVersions(d?.versions || [])).catch(() => {});
    };

    const handleDownload = async (version) => {
        setBusyVersion(prev => ({...prev, [version]: "downloading"}));
        try {
            await serverResource.installVersion(version);
        } finally {
            serverResource.installedVersions().then(d => setInstalledVersions(d?.versions || [])).catch(() => {});
            serverResource.downloadedVersions().then(d => setDownloadedVersions(d?.versions || [])).catch(() => {});
            setBusyVersion(prev => ({...prev, [version]: null}));
        }
    };

    const handleDeleteInstalled = async (version) => {
        setBusyVersion(prev => ({...prev, [version]: "deleting_installed"}));
        try {
            await serverResource.deleteInstalledVersion(version);
            setInstalledVersions(prev => prev.filter(v => v !== version));
        } finally {
            setBusyVersion(prev => ({...prev, [version]: null}));
            setExpandedDelete(false);
        }
    };

    const handleDeleteAll = async (version, resolvedKey) => {
        setBusyVersion(prev => ({...prev, [version]: "deleting_all"}));
        try {
            await Promise.all([
                serverResource.deleteDownload(resolvedKey).catch(() => {}),
                serverResource.deleteInstalledVersion(resolvedKey).catch(() => {}),
            ]);
            setDownloadedVersions(prev => prev.filter(v => v !== resolvedKey));
            setInstalledVersions(prev => prev.filter(v => v !== resolvedKey));
        } finally {
            setBusyVersion(prev => ({...prev, [version]: null}));
            setExpandedDelete(false);
            setConfirmDeleteAll(false);
        }
    };

    const handleDeleteDownload = async (version) => {
        setBusyVersion(prev => ({...prev, [version]: "deleting"}));
        try {
            await serverResource.deleteDownload(version);
            setDownloadedVersions(prev => prev.filter(v => v !== version));
        } finally {
            setBusyVersion(prev => ({...prev, [version]: null}));
        }
    };

    const handleCreateWithVersion = async (version) => {
        const payload = {
            ...createForm,
            version,
            port: createForm.port ? parseInt(createForm.port) : 0,
        };
        setBusyVersion(prev => ({...prev, [version]: "creating"}));
        try {
            await serverResource.create(payload);
            setCreateForm(emptyCreate);
            await refreshServers();
            await fetchNextPreview();
            serverResource.installedVersions().then(d => setInstalledVersions(d?.versions || [])).catch(() => {});
            serverResource.downloadedVersions().then(d => setDownloadedVersions(d?.versions || [])).catch(() => {});
        } finally {
            setBusyVersion(prev => ({...prev, [version]: null}));
        }
    };

    return (
        <>
            <Panel
                className="mb-6"
                title={t("servers.create", "Create server")}
                content={
                    <div>
                        {(() => {
                            const allVersions = [
                                {key: "stable", label: versionLabel("stable")},
                                {key: "experimental", label: versionLabel("experimental")},
                                ...installedVersions
                                    .filter(v => v !== availableVersions?.stable?.headless && v !== availableVersions?.experimental?.headless)
                                    .map(v => ({key: v, label: v})),
                                {key: "__other__", label: t("controls.other_version", "other...")}
                            ];
                            const isOther = createForm.version === "__other__";
                            const selKey = isOther ? (createForm.customVersion || "") : (createForm.version || "stable");
                            const resolvedKey = selKey === "stable"
                                ? availableVersions?.stable?.headless || selKey
                                : selKey === "experimental"
                                    ? availableVersions?.experimental?.headless || selKey
                                    : selKey;
                            const isInstalled = selKey === "stable"
                                ? installedVersions.includes(availableVersions?.stable?.headless)
                                : selKey === "experimental"
                                    ? installedVersions.includes(availableVersions?.experimental?.headless)
                                    : installedVersions.includes(selKey);
                            const isDownloaded = selKey === "stable"
                                ? downloadedVersions.includes(availableVersions?.stable?.headless)
                                : selKey === "experimental"
                                    ? downloadedVersions.includes(availableVersions?.experimental?.headless)
                                    : downloadedVersions.includes(selKey);
                            const isBusy = !!busyVersion[selKey] || (isOther && !selKey);
                            let badgeClass = "text-xs px-2 py-1 ";
                            let badgeText = "";
                            if (isInstalled && isDownloaded) { badgeClass += "text-green"; badgeText = "✓ installed · zip cached"; }
                            else if (isInstalled && !isDownloaded) { badgeClass += "text-green"; badgeText = "✓ installed"; }
                            else if (!isInstalled && isDownloaded) { badgeClass += "text-orange"; badgeText = "zip cached · not installed"; }
                            else { badgeClass += "text-gray-400"; badgeText = "not installed · no zip"; }
                            return (
                                <div className="grid grid-cols-1 xl:grid-cols-2 gap-2">
                                    <div className="flex gap-2">
                                        <input
                                            className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                                            placeholder={nextPreview.name || t("name")}
                                            value={createForm.name}
                                            onChange={e => setCreateForm({...createForm, name: e.target.value})}
                                        />
                                        <input
                                            className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                                            value={createForm.bind_ip}
                                            onChange={e => setCreateForm({...createForm, bind_ip: e.target.value})}
                                        />
                                        <input
                                            className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                                            placeholder={nextPreview.port ? String(nextPreview.port) : t("controls.port")}
                                            type="number"
                                            min={1}
                                            max={65535}
                                            value={createForm.port}
                                            onChange={e => setCreateForm({...createForm, port: e.target.value})}
                                        />
                                    </div>
                                    <div className="flex flex-wrap items-center gap-2">
                                    <select
                                        className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                                        value={createForm.version}
                                        onChange={e => setCreateForm({...createForm, version: e.target.value, customVersion: ""})}
                                    >
                                        {allVersions.map(({key, label}) => (
                                            <option key={key} value={key}>{label}</option>
                                        ))}
                                    </select>
                                    {isOther && (
                                        <input
                                            className="shadow appearance-none border py-2 px-3 text-black"
                                            style={{width: "100px", flexShrink: 0}}
                                            placeholder="2.0.76"
                                            value={createForm.customVersion || ""}
                                            title={t("controls.other_version_hint", "Enter a specific Factorio version number, e.g. 2.0.76. Full list: factorio.com/download/archive")}
                                            onChange={e => setCreateForm({...createForm, customVersion: e.target.value})}
                                            onBlur={e => {
                                                let v = e.target.value.trim();
                                                if (!v) return;
                                                v = v.replace(/[,\s]+/g, ".").replace(/\.{2,}/g, ".");
                                                if (/^\d+$/.test(v)) {
                                                    if (v.length <= 2) v = "2.0." + v;
                                                    else if (v.length === 3) v = v[0] + "." + v[1] + "." + v[2];
                                                    else v = v[0] + "." + v[1] + "." + v.slice(2);
                                                }
                                                const parts = v.split(".");
                                                if (parts.length === 2) v = parts[0] + ".0." + parts[1];
                                                const stableV = availableVersions?.stable?.headless;
                                                const expV = availableVersions?.experimental?.headless;
                                                if (stableV && v.startsWith(stableV)) { setCreateForm(f => ({...f, version: "stable", customVersion: ""})); return; }
                                                if (expV && v.startsWith(expV)) { setCreateForm(f => ({...f, version: "experimental", customVersion: ""})); return; }
                                                setCreateForm(f => ({...f, customVersion: v}));
                                            }}
                                        />
                                    )}
                                    <span className={badgeClass}>{badgeText}</span>
                                    {(!isInstalled || !isDownloaded) && !expandedDelete && (
                                        <Button size="sm" type="default" isLoading={busyVersion[selKey] === "downloading"} isDisabled={isBusy} onClick={() => handleDownload(selKey)}>
                                            {t("controls.download", "Download")}
                                        </Button>
                                    )}
                                    {!expandedDelete && (
                                    <Button size="sm" type="success" isLoading={busyVersion[selKey] === "creating"} isDisabled={isBusy} onClick={() => handleCreateWithVersion(selKey)}>
                                        {t("create")}
                                    </Button>
                                    )}
                                    {(isDownloaded || isInstalled) && (() => {
                                        const expanded = expandedDelete === selKey;
                                        const confirming = confirmDeleteAll === selKey;
                                        if (!expanded) return (
                                            <Button size="sm" type="danger" isDisabled={isBusy} onClick={() => { setExpandedDelete(selKey); setConfirmDeleteAll(false); }}>
                                                <FontAwesomeIcon icon={faTrash}/>
                                            </Button>
                                        );
                                        if (confirming) return (
                                            <>
                                                <Button size="sm" type="danger" isLoading={busyVersion[selKey] === "deleting_all"} onClick={() => handleDeleteAll(selKey, resolvedKey)}>
                                                    {t("servers.confirm_delete", "Confirm")}
                                                </Button>
                                                <Button size="sm" type="default" isDisabled={isBusy} onClick={() => { setExpandedDelete(false); setConfirmDeleteAll(false); }}>
                                                    ✕
                                                </Button>
                                            </>
                                        );
                                        return (
                                            <>
                                                {isDownloaded && (
                                                    <Button size="sm" type="danger" isLoading={busyVersion[selKey] === "deleting"} isDisabled={isBusy} onClick={() => handleDeleteDownload(resolvedKey)}>
                                                        zip
                                                    </Button>
                                                )}
                                                {isInstalled && (
                                                    <Button size="sm" type="danger" isLoading={busyVersion[selKey] === "deleting_installed"} isDisabled={isBusy} onClick={() => handleDeleteInstalled(resolvedKey)}>
                                                        installed
                                                    </Button>
                                                )}
                                                {isDownloaded && isInstalled && (
                                                    <Button size="sm" type="danger" isDisabled={isBusy} onClick={() => setConfirmDeleteAll(selKey)}>
                                                        all
                                                    </Button>
                                                )}
                                                <Button size="sm" type="default" isDisabled={isBusy} onClick={() => setExpandedDelete(false)}>
                                                    ✕
                                                </Button>
                                            </>
                                        );
                                    })()}
                                    </div>
                                </div>
                            );
                        })()}
                    </div>
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
                        versionLabel={versionLabel}
                        availableVersions={availableVersions}
                        installedVersions={installedVersions}
                        onUpdated={refreshServers}
                        t={t}
                    />
                ))}
            </div>
        </>
    );
};

const ServerCard = ({server, saves, selectedSave, setSelectedSave, busy, runAction, versionLabel, availableVersions, installedVersions, onUpdated, t}) => {
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
    const [ipDraft, setIpDraft] = useState(server.bindip || server.bind_ip || "0.0.0.0");
    const [portDraft, setPortDraft] = useState(server.port || "");
    const [fieldBusy, setFieldBusy] = useState(false);

    const resolveChannel = (srv) => {
        if (srv.version_channel === "stable" || srv.version_channel === "experimental") return srv.version_channel;
        const v = srv.version || "";
        if (v === "stable" || v === "experimental") return v;
        const stableV = availableVersions?.stable?.headless || "";
        const expV = availableVersions?.experimental?.headless || "";
        if (stableV && v.startsWith(stableV)) return "stable";
        if (expV && v.startsWith(expV)) return "experimental";
        return "stable";
    };

    const [versionDraft, setVersionDraft] = useState(() => resolveChannel(server));
    const versionUserEdited = React.useRef(false);

    useEffect(() => {
        setIpDraft(server.bindip || server.bind_ip || "0.0.0.0");
        setPortDraft(server.port || "");
        if (!versionUserEdited.current) {
            setVersionDraft(resolveChannel(server));
        }
    }, [server.bindip, server.bind_ip, server.port, server.version, server.fac_version]);

    const running = !!server.running;
    const starting = running && !server.rcon_connected;
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

    const saveField = async (field, value, revert) => {
        setFieldBusy(true);
        try {
            await serverResource.update(server.id, {[field]: value});
            await onUpdated();
        } catch (err) {
            const message = err?.response?.data || err?.message || "Update failed";
            if (window.flash) window.flash(String(message), "red");
            revert();
        } finally {
            setFieldBusy(false);
        }
    };

    const handleIpBlur = () => {
        const trimmed = ipDraft.trim();
        const current = server.bindip || server.bind_ip || "0.0.0.0";
        if (trimmed && trimmed !== current) {
            saveField("bind_ip", trimmed, () => setIpDraft(current));
        } else {
            setIpDraft(current);
        }
    };

    const handlePortBlur = () => {
        const parsed = parseInt(portDraft);
        if (parsed && parsed !== server.port) {
            saveField("port", parsed, () => setPortDraft(server.port || ""));
        } else {
            setPortDraft(server.port || "");
        }
    };

    const handleVersionChange = (e) => {
        const value = e.target.value;
        versionUserEdited.current = true;
        setVersionDraft(value);
        saveField("version", value, () => {
            versionUserEdited.current = false;
            setVersionDraft(server.version_channel || "stable");
        });
    };

    return (
        <div className="bg-gray-dark accentuated p-4">
            <div className="flex items-center justify-between gap-3 mb-4">
                <div className="flex items-center gap-3">
                    <h2 className="text-dirty-white text-xl font-bold">{server.name || `Server ${server.id}`}</h2>
                    <div className={starting ? "text-orange font-bold" : running ? "text-green font-bold" : "text-red font-bold"}>
                        {starting ? t("controls.starting", "Starting...") : running ? t("controls.running") : t("controls.stopped")}
                        {server.pending_restart ? <span className="ml-2 text-orange">({t("servers.pending_restart", "restart pending")})</span> : null}
                    </div>
                </div>

                <span className={`inline-block ${deleteDisabledReason ? "cursor-not-allowed" : ""}`} title={deleteDisabledReason || undefined}>
                    <Button
                        className={deleteDisabledReason ? "pointer-events-none" : ""}
                        size="sm"
                        type="danger"
                        isDisabled={!!deleteDisabledReason}
                        isLoading={busy[`${server.id}:delete`]}
                        onClick={deleteServer}
                    >
                        {isConfirmingDelete ? t("servers.confirm_delete", "Confirm delete") : <FontAwesomeIcon icon={faTrash}/>}
                    </Button>
                </span>
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
                <div>
                    <div className="font-bold mb-1">IP</div>
                    <input
                        className="shadow appearance-none border w-full py-1 px-2 text-black text-sm"
                        value={ipDraft}
                        disabled={fieldBusy}
                        onChange={e => setIpDraft(e.target.value)}
                        onBlur={handleIpBlur}
                    />
                </div>
                <div>
                    <div className="font-bold mb-1">{t("controls.port")}</div>
                    <input
                        type="number"
                        min={1}
                        max={65535}
                        className="shadow appearance-none border w-full py-1 px-2 text-black text-sm"
                        value={portDraft}
                        disabled={fieldBusy}
                        onChange={e => setPortDraft(e.target.value)}
                        onBlur={handlePortBlur}
                    />
                </div>
                <div>
                    <div className="font-bold mb-1">{t("controls.f_version")}</div>
                    <select
                        className="shadow appearance-none border w-full py-1 px-2 text-black text-sm"
                        value={versionDraft}
                        disabled={fieldBusy}
                        onChange={handleVersionChange}
                    >
                        <option value="stable">{versionLabel("stable")}</option>
                        <option value="experimental">{versionLabel("experimental")}</option>
                        {installedVersions.filter(v => v !== availableVersions?.stable?.headless && v !== availableVersions?.experimental?.headless).map(v => (
                            <option key={v} value={v}>{v}</option>
                        ))}
                    </select>
                </div>
                <div>
                    <div className="font-bold mb-1">{t("controls.save")}</div>
                    <select
                        className="shadow appearance-none border w-full py-1 px-2 text-black text-sm"
                        value={running ? (server.savefile || "") : selectedSave}
                        disabled={noSave || running}
                        onChange={e => setSelectedSave(e.target.value)}
                    >
                        {noSave
                            ? <option value="">{t("saves.upload_or_create", "Upload or create a save first")}</option>
                            : saves.map(save => <option key={save.name} value={save.name}>{save.name}</option>)
                        }
                    </select>
                </div>
            </div>

            <div className="flex gap-2 mb-3 items-stretch">
                {running ? (
                    <>
                        <Button className="flex-1" size="sm" type="default" isLoading={busy[`${server.id}:save`]} onClick={() => runAction(server, "save", () => serverResource.save(server.id))}>
                            {t("servers.save_now", "Save")}
                        </Button>
                        <Button className="flex-1" size="sm" type="default" isLoading={busy[`${server.id}:stop`]} onClick={() => runAction(server, "stop", () => serverResource.stop(server.id))}>
                            {t("controls.save&stop")}
                        </Button>
                        <Button className="flex-1" size="sm" type="danger" isLoading={busy[`${server.id}:kill`]} onClick={() => runAction(server, "kill", () => serverResource.kill(server.id))}>
                            {t("controls.kill_server")}
                        </Button>
                    </>
                ) : (
                    <span className={`inline-block w-full ${startDisabledReason ? "cursor-not-allowed" : ""}`} title={startDisabledReason || undefined}>
                        <Button
                            className={`w-full ${startDisabledReason ? "pointer-events-none" : ""}`}
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
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange accentuated text-black font-bold text-center" to={`/servers/${server.id}/saves`}>
                    {t("saves.title")}
                </Link>
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange accentuated text-black font-bold text-center" to={`/servers/${server.id}/mods`}>
                    {t("mods.title")}
                </Link>
                <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange accentuated text-black font-bold text-center" to={`/servers/${server.id}/server-settings`}>
                    {t("server_settings.title")}
                </Link>
                {running ? (
                    <Link className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange accentuated text-black font-bold text-center" to={`/servers/${server.id}/logs`}>
                        {t("console.title")}
                    </Link>
                ) : (
                    <span className="bg-gray-light py-1 px-2 accentuated text-black font-bold text-center opacity-40 cursor-not-allowed">
                        {t("console.title")}
                    </span>
                )}
            </div>
        </div>
    );
};

export default Controls;
