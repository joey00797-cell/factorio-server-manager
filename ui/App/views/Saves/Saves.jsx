import React, {useEffect, useState} from "react";
import savesResource from "../../../api/resources/saves";
import Panel from "../../components/Panel";
import CreateSaveForm from "./components/CreateSaveForm";
import UploadSaveForm from "./components/UploadSaveForm";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faDownload, faTrashAlt, faPlay, faSync, faUpload, faLock, faLockOpen, faPencilAlt, faCheck, faTimes} from "@fortawesome/free-solid-svg-icons";
import Button from "../../components/Button";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import serverResource from "../../../api/resources/server";
import ServerScopeHeader from "../../components/ServerScopeHeader";

const Saves = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();

    const [saves, setSaves] = useState([]);
    const [sortField, setSortField] = useState("last_mod");
    const [sortAsc, setSortAsc] = useState(false);
    const [serverStatus, setServerStatus] = useState({running: false});
    const [backups, setBackups] = useState([]);
    const [backupSchedule, setBackupSchedule] = useState({enabled: false, interval_minutes: 60, retention: 5});
    const [isRunningBackup, setIsRunningBackup] = useState(false);
    const [isSavingSchedule, setIsSavingSchedule] = useState(false);
    const [servers, setServers] = useState([]);
    const [isRestoring, setIsRestoring] = useState(false);
    const [restoreDropdown, setRestoreDropdown] = useState(null); // backup name
    const [editingBackup, setEditingBackup] = useState(null); // {name, value}
    const [deleteConfirm, setDeleteConfirm] = useState(null); // backup name

    useEffect(() => {
        if (!restoreDropdown) return;
        const handler = () => setRestoreDropdown(null);
        document.addEventListener('click', handler);
        return () => document.removeEventListener('click', handler);
    }, [restoreDropdown]);

    const updateList = () => {
        savesResource.list(false, serverId)
            .then(res => {
                if (res) {
                    setSaves(res);
                }
            })
    }

    const loadServers = () => {
        serverResource.list().then(res => setServers(res || [])).catch(() => {});
    }

    const loadBackups = () => {
        savesResource.backups.listAll(serverId).then(res => setBackups(res || [])).catch(() => {});
        savesResource.backups.schedule(serverId).then(setBackupSchedule).catch(() => {});
    }

    const runBackup = () => {
        setIsRunningBackup(true);
        savesResource.backups.run(serverId)
            .then(res => {
                loadBackups();
                if (res && res.length > 0) {
                    window.flash(t("saves.backup_done", "Backup completed") + ` (${res.length})`, "green");
                } else {
                    window.flash(t("saves.backup_skipped", "No changes, backup skipped"), "orange");
                }
            })
            .catch(() => window.flash(t("saves.backup_error", "Backup failed"), "red"))
            .finally(() => setIsRunningBackup(false));
    }

    const togglePin = (name) => {
        savesResource.backups.pin(serverId, name)
            .then(() => loadBackups())
            .catch(() => window.flash(t("saves.pin_error", "Failed to pin backup"), "red"));
    }

    const startRename = (b) => {
        setEditingBackup({name: b.name, value: b.name});
    }

    const confirmRename = () => {
        if (!editingBackup || editingBackup.value === editingBackup.name) {
            setEditingBackup(null);
            return;
        }
        savesResource.backups.rename(serverId, editingBackup.name, editingBackup.value)
            .then(() => { loadBackups(); setEditingBackup(null); window.flash(t("saves.rename_done", "Backup renamed"), "green"); })
            .catch(() => window.flash(t("saves.rename_error", "Failed to rename backup"), "red"));
    }

    const doRestore = (name, targetServerId, sourceServerId) => {
        setIsRestoring(true);
        setRestoreDropdown(null);
        savesResource.backups.restore(serverId, name, targetServerId, sourceServerId)
            .then(() => { updateList(); window.flash(t("saves.backup_restored", "Backup restored to saves"), "green"); })
            .catch(() => window.flash(t("saves.backup_restore_error", "Failed to restore backup"), "red"))
            .finally(() => setIsRestoring(false));
    }

    const deleteBackup = (name) => {
        savesResource.backups.remove(serverId, name)
            .then(() => loadBackups())
            .catch(() => window.flash(t("saves.backup_delete_error", "Failed to delete backup"), "red"));
    }

    const saveSchedule = () => {
        setIsSavingSchedule(true);
        savesResource.backups.updateSchedule(serverId, {
            ...backupSchedule,
            interval_minutes: parseInt(backupSchedule.interval_minutes) || 60,
            retention: parseInt(backupSchedule.retention) || 5,
        }).then(res => { setBackupSchedule(res); window.flash(t("saves.schedule_saved", "Backup schedule saved"), "green"); })
          .catch(() => window.flash(t("saves.schedule_error", "Failed to save schedule"), "red"))
          .finally(() => setIsSavingSchedule(false));
    }

    useEffect(() => {
        updateList();
        loadBackups();
        loadServers();
        serverResource.status(serverId).then(setServerStatus);
    }, [serverId]);

    const deleteSave = async (save) => {
        const res = await savesResource.delete(save, serverId);
        if (res) {
            updateList()
        }
    }

    return (
        <>
            <ServerScopeHeader/>
            <div className="lg:flex mb-6">
                <Panel
                    title={t("saves.create_save")}
                    className="lg:w-1/2 lg:mr-3 mb-6 lg:mb-0"
                    content={
                        serverStatus.running
                            ? <p className="text-red-light pt-4 pb-24">
                                {t("saves.create_new_save_only_when_server_not_running")}
                            </p>
                            : <CreateSaveForm onSuccess={updateList} serverId={serverId}/>
                    }
                />
                <Panel
                    title={t("saves.upload_save")}
                    className="lg:w-1/2 lg:ml-3"
                    content={<UploadSaveForm onSuccess={updateList} serverId={serverId}/>}
                />
            </div>

            <Panel
                className="mb-4"
                title={t("saves.title")}
                content={
                    <div className="overflow-x-auto w-full">
                        <table className="w-full">
                            <thead>
                            <tr className="text-left py-1">
                                {[["name", t("name")], ["last_mod", t("saves.last_modified")], ["size", t("saves.size")]].map(([field, label]) => (
                                    <th key={field} className="cursor-pointer select-none pr-4 hover:text-orange" onClick={() => { if (sortField === field) setSortAsc(a => !a); else { setSortField(field); setSortAsc(true); } }}>
                                        {label}<span className="text-gray-400 text-xs ml-1">{sortField === field ? (sortAsc ? "▲" : "▼") : "↕"}</span>
                                    </th>
                                ))}
                                <th>{t("actions")}</th>
                            </tr>
                            </thead>
                            <tbody>
                            {saves.length === 0 && (
                                <tr>
                                    <td className="py-4 text-gray-light" colSpan="4">
                                        {t("saves.empty", "No saves for this server yet.")}
                                    </td>
                                </tr>
                            )}
                            {saves.slice().sort((a, b) => {
                                let av = sortField === "size" ? a.size : sortField === "last_mod" ? new Date(a.last_mod) : a.name.toLowerCase();
                                let bv = sortField === "size" ? b.size : sortField === "last_mod" ? new Date(b.last_mod) : b.name.toLowerCase();
                                if (av < bv) return sortAsc ? -1 : 1;
                                if (av > bv) return sortAsc ? 1 : -1;
                                return 0;
                            }).map(save =>
                                <tr className="py-2 md:py-1 hover:glow-orange hover:bg-orange hover:text-black cursor-pointer" key={save.name}>
                                    <td className="pr-4">{save.name}</td>
                                    <td className="pr-4">{(new Date(save.last_mod)).toLocaleString()}</td>
                                    <td className="pr-4">{parseFloat(save.size / 1024 / 1024).toFixed(3)} MB</td>
                                    <td>
                                        <a href={savesResource.downloadURL(save.name, serverId)} className="mr-2">
                                            <FontAwesomeIcon
                                                className="text-gray-light cursor-pointer hover:text-orange"
                                                icon={faDownload}/>
                                        </a>
                                        <FontAwesomeIcon className="text-red cursor-pointer hover:text-red-light mr-2"
                                                         onClick={() => deleteSave(save)} icon={faTrashAlt}/>
                                    </td>
                                </tr>
                            )}
                            </tbody>
                        </table>
                    </div>
                }
            />
        <Panel
            title={t("saves.backups_title", "Save Backups")}
            content={
                <div>
                    <div className="flex gap-4 mb-4 flex-wrap items-end">
                        <div>
                            <label className="block text-gray-500 font-bold text-sm mb-1">{t("saves.backup_enabled", "Enabled")}</label>
                            <input type="checkbox" checked={backupSchedule.enabled}
                                onChange={e => setBackupSchedule({...backupSchedule, enabled: e.target.checked})}/>
                        </div>
                        <div>
                            <label className="block text-gray-500 font-bold text-sm mb-1">{t("saves.backup_interval", "Interval (min)")}</label>
                            <input className="shadow appearance-none border w-24 py-2 px-3 text-black"
                                type="number" min="1"
                                value={backupSchedule.interval_minutes}
                                onChange={e => setBackupSchedule({...backupSchedule, interval_minutes: e.target.value})}/>
                        </div>
                        <div>
                            <label className="block text-gray-500 font-bold text-sm mb-1">{t("saves.backup_retention", "Retention")}</label>
                            <input className="shadow appearance-none border w-24 py-2 px-3 text-black"
                                type="number" min="1"
                                value={backupSchedule.retention}
                                onChange={e => setBackupSchedule({...backupSchedule, retention: e.target.value})}/>
                        </div>
                        <div className="flex gap-2 shrink-0">
                            <Button type="success" isLoading={isSavingSchedule} onClick={saveSchedule}>{t("save")}</Button>
                            <Button type="default" isLoading={isRunningBackup} onClick={runBackup}>
                                <FontAwesomeIcon icon={faSync} className="mr-1"/>{t("saves.backup_run_now", "Run now")}
                            </Button>
                        </div>
                    </div>
                    {backups.length === 0 ? (
                        <div className="text-gray-500 italic">{t("saves.no_backups", "No backups yet")}</div>
                    ) : (
                        <table className="w-full">
                            <thead><tr>
                                <th className="text-left pr-4">{t("name")}</th>
                                <th className="text-left pr-4">{t("saves.size")}</th>
                                <th className="text-left pr-4">{t("saves.date")}</th>
                                <th/>
                            </tr></thead>
                            <tbody>
                            {backups.map(b => {
                                const isForeign = b.source_server_id && b.source_server_id !== serverId;
                                return (
                                <tr key={b.source_server_id + ":" + b.name} className={`py-1 hover:glow-orange hover:bg-orange hover:text-black group${isForeign ? " opacity-50" : ""}`}>
                                    <td className="pr-4">
                                        {isForeign && <span className="text-xs text-gray-400 mr-1">[{b.source_server_name}]</span>}
                                        {editingBackup && editingBackup.name === b.name ? (
                                            <span className="flex items-center gap-1">
                                                <input className="border px-1 py-0 text-black text-xs"
                                                    value={editingBackup.value}
                                                    onChange={e => setEditingBackup({...editingBackup, value: e.target.value})}
                                                    onKeyDown={e => { if (e.key === 'Enter') confirmRename(); if (e.key === 'Escape') setEditingBackup(null); }}
                                                    autoFocus/>
                                                <FontAwesomeIcon className="text-green cursor-pointer" icon={faCheck} onClick={confirmRename}/>
                                                <FontAwesomeIcon className="text-red cursor-pointer" icon={faTimes} onClick={() => setEditingBackup(null)}/>
                                            </span>
                                        ) : (
                                            <span className={b.pinned ? "font-bold" : ""}>{b.name}</span>
                                        )}
                                    </td>
                                    <td className="pr-4">{parseFloat(b.size / 1024 / 1024).toFixed(3)} MB</td>
                                    <td className="pr-4">{new Date(b.created_at).toLocaleString()}</td>
                                    <td>
                                        <span className="relative inline-block mr-2">
                                            <FontAwesomeIcon className={`cursor-pointer mr-2 group-hover:text-black ${b.pinned ? "text-orange" : "text-gray-500"}`}
                                                icon={b.pinned ? faLock : faLockOpen} onClick={() => togglePin(b.name)} title={t("saves.pin", "Pin")}/>
                                            <FontAwesomeIcon className="text-gray-light cursor-pointer group-hover:text-black mr-2"
                                                icon={faPencilAlt} onClick={() => startRename(b)} title={t("saves.rename", "Rename")}/>
                                            <FontAwesomeIcon className={`cursor-pointer group-hover:text-black mr-2 ${isForeign ? "text-green" : "text-gray-light"}`}
                                                onClick={e => { e.stopPropagation(); setRestoreDropdown(restoreDropdown === b.name ? null : b.name); }} icon={faUpload} title={t("saves.backup_restore", "Restore")}/>
                                            {restoreDropdown === b.name && (
                                                <div className="absolute left-0 bottom-6 bg-white border border-gray-300 z-50 shadow-lg" style={{minWidth: "120px"}}>
                                                    {servers.length === 1
                                                        ? <div className="px-2 py-1 cursor-pointer hover:bg-orange text-black text-xs"
                                                            onClick={() => doRestore(b.name, servers[0].id, b.source_server_id)}>
                                                            {t("saves.restore_here", "Restore here")}
                                                          </div>
                                                        : servers.map(s => (
                                                            <div key={s.id} className="px-2 py-1 cursor-pointer hover:bg-orange text-black text-xs"
                                                                onClick={() => doRestore(b.name, s.id, b.source_server_id)}>
                                                                {s.name || `Server ${s.id}`}
                                                            </div>
                                                        ))
                                                    }
                                                </div>
                                            )}
                                        </span>
                                        <a href={savesResource.backups.downloadUrl(serverId, b.name)} className="mr-2" title={t("saves.download", "Download")}>
                                            <FontAwesomeIcon className="text-gray-light cursor-pointer group-hover:text-black" icon={faDownload}/>
                                        </a>
                                        {deleteConfirm === b.name
                                            ? <span className="inline-flex items-center gap-1">
                                                <span className="text-xs text-red">{isForeign ? `[${b.source_server_name}] ` : ""}{t("saves.confirm_delete", "Delete?")}</span>
                                                <FontAwesomeIcon className="text-red cursor-pointer" icon={faCheck} onClick={() => { setDeleteConfirm(null); deleteBackup(b.name); }}/>
                                                <FontAwesomeIcon className="text-gray-light cursor-pointer" icon={faTimes} onClick={() => setDeleteConfirm(null)}/>
                                              </span>
                                            : <FontAwesomeIcon className="text-red cursor-pointer group-hover:text-black" title={t("saves.delete", "Delete")}
                                                onClick={() => setDeleteConfirm(b.name)} icon={faTrashAlt}/>}
                                    </td>
                                </tr>
                            );
                            })}
                            </tbody>
                        </table>
                    )}
                </div>
            }
        />

    </>
    )
}

export default Saves;
