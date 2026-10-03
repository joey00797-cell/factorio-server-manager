import React, {useEffect, useRef, useState} from "react";
import Button from "../../../components/Button";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faExternalLinkAlt, faUpload, faSave, faGlobe} from "@fortawesome/free-solid-svg-icons";
import modLibrary from "../../../../api/resources/modLibrary";
import modsResource from "../../../../api/resources/mods";
import savesResource from "../../../../api/resources/saves";
import {useTranslation} from "react-i18next";
import SelectVersionForm from "./AddMod/components/SelectVersionForm";
import FactorioLogin from "./AddMod/components/FactorioLogin";
import Fuse from "fuse.js";
import socket from "../../../../api/socket";

const ModLibrary = ({serverId, onModUploaded, onApplied, preview, libraryMods = []}) => {
    const {t} = useTranslation();
    const [isApplying, setIsApplying] = useState(false);
    const [syncMismatched, setSyncMismatched] = useState([]);
    const [applyOverrides, setApplyOverrides] = useState({});

    // Upload ZIP
    const [uploadFile, setUploadFile] = useState(null);
    const [isUploading, setIsUploading] = useState(false);

    // Load from Save
    const [saves, setSaves] = useState([]);
    const [selectedSave, setSelectedSave] = useState("");
    const [isSyncing, setIsSyncing] = useState(false);
    const [isSyncingDeps, setIsSyncingDeps] = useState(false);
    const [syncDropdown, setSyncDropdown] = useState(false);
    const [syncCurrentMode, setSyncCurrentMode] = useState("full");
    const syncCurrentModeRef = useRef("full");

    // Portal
    const [isPortalAuth, setIsPortalAuth] = useState(false);
    const [portalUsername, setPortalUsername] = useState("");
    const [portalModList, setPortalModList] = useState([]);
    const [fuse, setFuse] = useState(null);
    const [search, setSearch] = useState("");
    const [suggested, setSuggested] = useState([]);
    const [selectedMod, setSelectedMod] = useState(null);
    const [releases, setReleases] = useState([]);
    const [isModalOpen, setIsModalOpen] = useState(false);
    const [isImporting, setIsImporting] = useState(false);
    const [autocomplete, setAutocomplete] = useState(NaN);

    useEffect(() => {
        modsResource.portal.status().then(status => {
            if (status && status.logged_in) {
                setIsPortalAuth(true);
                setPortalUsername(status.username || "");
            }
        });
        savesResource.list(false, serverId).then(s => setSaves(s || []));
    }, [serverId]);

    useEffect(() => {
        const room = serverId ? `servers:${serverId}:mods_sync` : 'mods_sync';
        console.log("Subscribing to room:", room, "serverId:", serverId, typeof serverId);
        const handler = (raw) => {
            const data = typeof raw === "string" ? JSON.parse(raw) : raw;
            if (data.status === "done") {
                setIsSyncing(false);
                const mismatched = (data.mods || []).filter(m => m.status === "version_mismatch");
                setSyncMismatched(mismatched);
                const doRefresh = (result) => {
                    try { if (onModUploaded) onModUploaded(); } catch(e) { console.error("onModUploaded error:", e); }
                    try { if (onApplied) onApplied(result); } catch(e) { console.error("onApplied error:", e); }
                };
                if (syncCurrentModeRef.current === "match") {
                    modLibrary.manifest.apply(serverId).then(doRefresh).catch(e => { console.error("apply error:", e); doRefresh(); });
                } else {
                    doRefresh();
                }
                if (mismatched.length === 0) window.flash(t("mods.sync_success", "Mods synced from save"), "green");
            } else if (data.status === "error") {
                setIsSyncing(false);
                window.flash(t("mods.sync_error", "Sync failed: ") + (data.message || ""), "red");
            }
        };
        const resubscribe = () => socket.emit('mods sync subscribe', serverId);
        socket.on(room, handler);
        socket.on("reconnect", resubscribe);
        socket.emit('mods sync subscribe', serverId);
        return () => {
            socket.off(room, handler);
            socket.off("reconnect", resubscribe);
            socket.emit('mods sync unsubscribe', serverId);
        };
    }, [serverId]);

    useEffect(() => {
        if (isPortalAuth) {
            modsResource.portal.list().then(data => {
                const mods = data?.results || data?.mods || data || [];
                setPortalModList(mods);
                setFuse(new Fuse(mods, {keys: ["title", "name"], threshold: 0.3}));
            }).catch(() => {});
        }
    }, [isPortalAuth]);

    const updateSuggested = (val) => {
        clearTimeout(autocomplete);
        setAutocomplete(setTimeout(() => {
            if (fuse) setSuggested(fuse.search(val || "").slice(0, 10));
        }, 200));
    };

    const selectMod = (mod) => {
        clearTimeout(autocomplete);
        setSearch(mod.item.title);
        setSuggested([]);
        setSelectedMod(mod);
    };

    const openVersionModal = async () => {
        const info = await modsResource.portal.info(selectedMod.item.name);
        setReleases(info.releases || []);
        setIsModalOpen(true);
    };

    const importRelease = async (release) => {
        setIsImporting(true);
        try {
            await modLibrary.portalImport(release.download_url, release.file_name, selectedMod.item.name);
            window.flash(t("mods.import_success", "Mod imported to library"), "green");
            setIsModalOpen(false);
            setSearch("");
            setSelectedMod(null);
            if (onModUploaded) onModUploaded();
        } catch (e) {
            window.flash(t("mods.import_error", "Import failed"), "red");
        } finally {
            setIsImporting(false);
        }
    };

    const logout = () => {
        modsResource.portal.logout().then(() => {
            setIsPortalAuth(false);
            setPortalUsername("");
            setPortalModList([]);
            setFuse(null);
        });
    };

    const uploadMod = async () => {
        if (!uploadFile) return;
        setIsUploading(true);
        try {
            await modLibrary.upload(uploadFile);
            setUploadFile(null);
            window.flash(t("mods.upload_success", "Mod uploaded to library"), "green");
            if (onModUploaded) onModUploaded();
        } catch (e) {
            window.flash(t("mods.upload_error", "Upload failed"), "red");
        } finally {
            setIsUploading(false);
        }
    };

    const syncFromSave = async (mode) => {
        if (!selectedSave) return;
        setSyncDropdown(false);
        setSyncCurrentMode(mode);
        syncCurrentModeRef.current = mode;
        setIsSyncing(true);
        try {
            await modsResource.syncFromSave(selectedSave, [], serverId, mode);
        } catch (e) {
            setIsSyncing(false);
            window.flash(t("mods.sync_error", "Sync failed"), "red");
        }
    };

    const syncDependencies = async () => {
        setIsSyncingDeps(true);
        try {
            const results = await modLibrary.manifest.syncDeps(serverId);
            const downloaded = (results || []).filter(r => r.status === "downloaded").length;
            const fromLib = (results || []).filter(r => r.status === "from_library").length;
            if (downloaded + fromLib > 0) {
                window.flash(t("mods.sync_deps_success", `Synced ${downloaded + fromLib} dependencies`), "green");
            } else {
                window.flash(t("mods.sync_deps_none", "All dependencies already satisfied"), "green");
            }
            if (onApplied) onApplied(null);
        } catch (e) {
            window.flash(t("mods.sync_deps_error", "Failed to sync dependencies"), "red");
        } finally {
            setIsSyncingDeps(false);
        }
    };

    const applyManifest = async () => {
        setIsApplying(true);
        try {
            // Apply overrides: update manifest to exclude unchecked items
            const currentItems = (await modLibrary.manifest.get(serverId))?.items || [];
            const toAdd = (preview.to_add || []).filter(m => applyOverrides[m.name] !== false);
            const toRemove = (preview.to_remove || []).filter(m => applyOverrides[m.name] !== false);
            const toEnable = (preview.to_enable || []).filter(m => applyOverrides[m.name] !== false);
            const toDisable = (preview.to_disable || []).filter(m => applyOverrides[m.name] !== false);
            const removeNames = new Set(toRemove.map(m => m.name));
            const disableNames = new Set(toDisable.map(m => m.name));
            const enableNames = new Set(toEnable.map(m => m.name));
            const addNames = new Set(toAdd.map(m => m.name));
            let newItems = currentItems
                .filter(i => !removeNames.has(i.asset?.name))
                .map(i => ({
                    asset_id: i.asset_id,
                    enabled: disableNames.has(i.asset?.name) ? false : enableNames.has(i.asset?.name) ? true : i.enabled
                }));
            // Add new items from to_add
            for (const m of toAdd) {
                const libMod = (libraryMods || []).find(lm => lm.name === m.name);
                if (libMod) newItems.push({asset_id: libMod.id, enabled: true});
            }
            if (Object.values(applyOverrides).some(v => !v)) {
                await modLibrary.manifest.update(serverId, newItems);
            }
            const result = await modLibrary.manifest.apply(serverId);
            window.flash(t("mods.apply_success", "Mods applied successfully"), "green");
            if (onApplied) onApplied(result);
        } catch (e) {
            window.flash(t("mods.apply_error", "Failed to apply mods"), "red");
        } finally {
            setIsApplying(false);
        }
    };

    const cancelChanges = async () => {
        try {
            const result = await modLibrary.manifest.reset(serverId);
            window.flash(t("mods.cancel_success", "Changes cancelled"), "blue");
            if (onApplied) onApplied(result);
        } catch (e) {
            window.flash(t("mods.cancel_error", "Failed to cancel changes"), "red");
        }
    };

    const hasPreview = preview && ((preview.to_add?.length || 0) + (preview.to_remove?.length || 0) + (preview.to_enable?.length || 0) + (preview.to_disable?.length || 0) > 0);
    React.useEffect(() => {
        if (!hasPreview) { setApplyOverrides({}); return; }
        const overrides = {};
        [...(preview.to_add || []), ...(preview.to_remove || []), ...(preview.to_enable || []), ...(preview.to_disable || [])].forEach(m => { overrides[m.name] = true; });
        setApplyOverrides(overrides);
    }, [hasPreview, preview]);

    return (
        <>
            {/* Three inline-expand cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4 items-center">

                {/* Load from Save */}
                <div className="flex items-center gap-2">
                    <select
                        className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                        value={selectedSave}
                        onChange={e => setSelectedSave(e.target.value)}
                    >
                        <option value="">{t("mods.select_save", "📁 Load from Save...")}</option>
                        {saves.map(s => <option key={s.name} value={s.name}>{s.name}</option>)}
                    </select>
                    {selectedSave && (
                        <span className="relative inline-block">
                            <Button size="sm" type="success" isLoading={isSyncing}
                                onClick={e => { e.stopPropagation(); setSyncDropdown(d => !d); }}>
                                {t("mods.sync", "Sync")}
                            </Button>
                            {syncDropdown && (
                                <div className="absolute right-0 top-full mt-1 bg-white border border-gray-300 z-50 shadow-lg" style={{minWidth: "180px"}}>
                                    <div className="px-3 py-2 cursor-pointer hover:bg-orange text-black text-sm"
                                        onClick={() => syncFromSave("match")}>
                                        Match save — enable save mods, disable others
                                    </div>
                                    <div className="px-3 py-2 cursor-pointer hover:bg-orange text-black text-sm"
                                        onClick={() => syncFromSave("add")}>
                                        Add from save — only enable save mods
                                    </div>
                                </div>
                            )}
                        </span>
                    )}
                </div>

                {/* Upload ZIP */}
                <div className="flex items-center gap-2">
                    <label className="shadow border py-2 px-3 text-black bg-white flex-1 min-w-0 cursor-pointer truncate">
                        {uploadFile ? uploadFile.name : <span className="opacity-60">{t("mods.upload_zip", "⬆ Upload ZIP...")}</span>}
                        <input type="file" accept=".zip" className="hidden" onChange={e => setUploadFile(e.target.files[0])}/>
                    </label>
                    {uploadFile && (
                        <Button size="sm" type="success" onClick={uploadMod} isLoading={isUploading}>
                            {t("mods.upload", "Upload")}
                        </Button>
                    )}
                </div>

                {/* Mod Portal */}
                <div className="flex items-center gap-2 min-w-0">
                    {!isPortalAuth ? (
                        <div className="flex-1 min-w-0 overflow-hidden">
                            <FactorioLogin setIsFactorioAuthenticated={setIsPortalAuth} setPortalUsername={setPortalUsername}/>
                        </div>
                    ) : (
                        <>
                            <div className="relative flex-1 min-w-0">
                                <input
                                    type="text"
                                    className="shadow appearance-none border w-full py-2 px-3 text-black"
                                    placeholder={t("mods.search_portal", "🌐 Search portal...")}
                                    value={search}
                                    onChange={e => { setSearch(e.target.value); setSelectedMod(null); updateSuggested(e.target.value); }}
                                />
                                {suggested.length > 0 && (
                                    <ul className="absolute z-50 bg-gray-dark border w-full max-h-48 overflow-y-auto">
                                        {suggested.map((mod, i) => (
                                            <li key={i} className="px-2 py-1 cursor-pointer hover:bg-orange hover:text-black" onClick={() => selectMod(mod)}>
                                                {mod.item.title}
                                            </li>
                                        ))}
                                    </ul>
                                )}
                            </div>
                            {selectedMod ? (
                                <Button onClick={openVersionModal} isLoading={isImporting}>
                                    {t("mods.import", "Import")}
                                </Button>
                            ) : (
                                <Button type="danger" onClick={logout}>{t("logout")}</Button>
                            )}
                        </>
                    )}
                </div>
            </div>

            {/* Version Mismatch */}
            {syncMismatched.length > 0 && (
                <div className="mb-4 p-3 bg-orange text-black">
                    <div className="font-bold text-sm mb-2">⚠ Version mismatch — library has different versions:</div>
                    {syncMismatched.map(m => (
                        <div key={m.name} className="text-sm flex justify-between">
                            <span>{m.name}</span>
                            <span>save: {m.version} · library: {m.available_version}</span>
                        </div>
                    ))}
                    <div className="text-xs mt-2 opacity-75">Log in to mod portal to download exact versions, or use library versions.</div>
                    <div className="flex gap-2 mt-3">
                        <button className="px-3 py-1 bg-black text-white text-xs hover:bg-gray-800"
                            onClick={() => setSyncMismatched([])}>
                            OK
                        </button>
                    </div>
                </div>
            )}

            {/* Apply Preview */}
            {hasPreview && (
                <div className="mb-4">
                    <div className="font-bold text-sm mb-2">{t("mods.apply_preview", "Apply Preview")}</div>
                    <div className="text-sm">
                        {preview.issues?.length > 0 && preview.issues.map((issue, i) => (
                            <div key={i} className="text-red mb-1">⚠ {issue.message}</div>
                        ))}
                        {preview.needs_server_stop && (
                            <div className="text-orange mb-2">{t("mods.needs_server_stop", "Server must be stopped before applying.")}</div>
                        )}
                        <div className="grid grid-cols-2 gap-4 mb-3">
                            <div>
                                <div className="font-bold text-green mb-1">{t("mods.to_add", "To Add")} ({preview.to_add?.length || 0})</div>
                                {(preview.to_add || []).map(m => (
                                    <div key={m.name} className="flex items-center gap-2">
                                        <input type="checkbox" checked={applyOverrides[m.name] !== false}
                                            onChange={e => setApplyOverrides(p => ({...p, [m.name]: e.target.checked}))}/>
                                        <span className={applyOverrides[m.name] === false ? "line-through opacity-50" : ""}>{m.name} {m.version}</span>
                                    </div>
                                ))}
                            </div>
                            <div>
                                <div className="font-bold text-red mb-1">{t("mods.to_remove", "To Remove")} ({preview.to_remove?.length || 0})</div>
                                {(preview.to_remove || []).map(m => (
                                    <div key={m.name} className="flex items-center gap-2">
                                        <input type="checkbox" checked={applyOverrides[m.name] !== false}
                                            onChange={e => setApplyOverrides(p => ({...p, [m.name]: e.target.checked}))}/>
                                        <span className={applyOverrides[m.name] === false ? "line-through opacity-50" : ""}>{m.name} {m.version}</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                        <div className="flex gap-2">
                            <Button size="sm" type="success" onClick={applyManifest} isLoading={isApplying} isDisabled={!preview.can_apply}>
                                {t("mods.apply_to_server", "Apply to Server")}
                            </Button>
                            {(preview.issues || []).some(i => i.code === "missing_dependency") && (() => {
                                const missingNames = (preview.issues || [])
                                    .filter(i => i.code === "missing_dependency")
                                    .map(i => i.message.match(/requires (.+)$/)?.[1])
                                    .filter(Boolean);
                                const allLocal = missingNames.every(name =>
                                    (libraryMods || []).some(m => m.name === name)
                                );
                                const canSync = allLocal || isPortalAuth;
                                return <Button size="sm" type="warning"
                                    onClick={canSync ? syncDependencies : () => window.flash(t("mods.login_first", "Login to portal first"), "red")}
                                    isLoading={isSyncingDeps}>
                                    {t("mods.sync_dependencies", "Sync Dependencies")}
                                </Button>;
                            })()}
                            <Button size="sm" type="danger" onClick={cancelChanges}>
                                {t("mods.cancel_changes", "Cancel")}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {isModalOpen && (
                <SelectVersionForm
                    releases={releases}
                    install={importRelease}
                    isOpen={isModalOpen}
                    setIsOpen={setIsModalOpen}
                    serverId={serverId}
                    factorioVersion={null}
                />
            )}
        </>
    );
};

export default ModLibrary;
