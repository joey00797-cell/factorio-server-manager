import React, {useEffect, useState} from "react";
import savesResource from "../../../../api/resources/saves";
import Label from "../../../components/Label";
import Button from "../../../components/Button";
import modsResource from "../../../../api/resources/mods";
import FactorioLogin from "./AddMod/components/FactorioLogin";
import socket from "../../../../api/socket";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {useTranslation} from "react-i18next";
import {faSpinner, faCheck, faTimes, faMinusCircle, faExternalLinkAlt} from "@fortawesome/free-solid-svg-icons";

const DLC_MODS = new Set(['elevated-rails', 'quality', 'space-age']);

const STATUS_ICON = {
    downloading:   <FontAwesomeIcon icon={faSpinner} spin={true} className="text-orange"/>,
    downloaded:    <FontAwesomeIcon icon={faCheck} className="text-green"/>,
    installed:     <FontAwesomeIcon icon={faCheck} className="text-green"/>,
    wrong_version: <FontAwesomeIcon icon={faCheck} className="text-yellow-500"/>,
    missing:       <FontAwesomeIcon icon={faTimes} className="text-red"/>,
    builtin:       <FontAwesomeIcon icon={faMinusCircle} className="text-blue-400"/>,
    not_found:     <FontAwesomeIcon icon={faTimes} className="text-red"/>,
};

const STATUS_TEXT = (t) => ({
    downloading:   t("mods.status_downloading"),
    downloaded:    t("mods.status_downloaded"),
    installed:     t("mods.status_already_installed"),
    wrong_version: t("mods.status_not_found"),
    missing:       t("mods.status_not_found"),
    builtin:       t("mods.status_builtin"),
    not_found:     t("mods.status_not_found"),
});

const LoadMods = ({refreshMods, serverId}) => {
    const {t} = useTranslation();
    const [saves, setSaves] = useState([]);
    const [selectedSave, setSelectedSave] = useState("");
    const [isLoading, setIsLoading] = useState(false);
    const [isSyncing, setIsSyncing] = useState(false);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isFactorioAuthenticated, setIsFactorioAuthenticated] = useState(false);
    const [modRows, setModRows] = useState([]);
    const [checkedMods, setCheckedMods] = useState({});
    const [syncError, setSyncError] = useState(null);
    const [currentMod, setCurrentMod] = useState(null);
    const [warning, setWarning] = useState(null);

    useEffect(() => {
        (async () => {
            setIsFactorioAuthenticated(await modsResource.portal.status());
            const s = await savesResource.list(false, serverId);
            setSaves(s);
            if (s.length > 0) {
                setIsDisabled(false);
                setSelectedSave(s[0].name);
            }
        })();
    }, []);

    useEffect(() => {
        const handler = (message) => {
            const data = JSON.parse(message);
            if (data.status === "progress") {
                setCurrentMod(data.mod);
                setModRows(rows => rows.map(r =>
                    r.name === data.mod ? {...r, status: "downloading"} : r
                ));
            } else if (data.status === "done") {
                setIsSyncing(false);
                setCurrentMod(null);
                setWarning(data.warning || null);
                if (data.mods) {
                    setModRows(data.mods);
                    setCheckedMods({});
                }
                refreshMods();
            } else if (data.status === "error") {
                setIsSyncing(false);
                setCurrentMod(null);
                setSyncError(data.message);
            }
        };

        const room = serverId ? `servers:${serverId}:mods_sync` : 'mods_sync';
        socket.on(room, handler);
        socket.emit('mods sync subscribe', serverId);
        return () => {
            socket.off(room, handler);
            socket.emit('mods sync unsubscribe', serverId);
        };
    }, [serverId]);

    const onReadSave = async () => {
        if (!selectedSave) return;
        setIsLoading(true);
        setModRows([]);
        setCheckedMods({});
        setSyncError(null);
        setWarning(null);

        try {
            const mods = await modsResource.getFromSave(selectedSave, serverId);
            setModRows(mods || []);
            // По умолчанию отмечаем missing и wrong_version
            const checked = {};
            (mods || []).forEach(m => {
                if (m.status === 'missing' || m.status === 'wrong_version') {
                    checked[m.name] = true;
                }
            });
            setCheckedMods(checked);
        } catch(e) {
            setSyncError("Failed to read save: " + e.message);
        } finally {
            setIsLoading(false);
        }
    };

    const onSync = async () => {
        const toSync = modRows
            .filter(m => checkedMods[m.name])
            .map(m => m.name);
        if (toSync.length === 0) return;

        setIsSyncing(true);
        setSyncError(null);
        try {
            const selectedModNames = Object.keys(checkedMods).filter(k => checkedMods[k]);
            await modsResource.syncFromSave(selectedSave, selectedModNames, serverId);
        } catch(e) {
            setIsSyncing(false);
            setSyncError("Failed to start sync: " + e.message);
        }
    };

    const toggleCheck = (name) => {
        setCheckedMods(prev => ({...prev, [name]: !prev[name]}));
    };

    const selectAll = () => {
        const checked = {};
        modRows.forEach(m => {
            if (m.status !== 'builtin' && m.status !== 'installed') {
                checked[m.name] = true;
            }
        });
        setCheckedMods(checked);
    };

    const clearAll = () => setCheckedMods({});

    const checkedCount = Object.values(checkedMods).filter(Boolean).length;

    if (!isFactorioAuthenticated) {
        return <FactorioLogin setIsFactorioAuthenticated={setIsFactorioAuthenticated}/>;
    }

    return (
        <div>
            {/* Выбор сейва */}
            <Label text={t("controls.save")} htmlFor="save"/>
            <select
                className="shadow appearance-none border w-full py-2 px-3 text-black mb-4"
                disabled={isDisabled}
                value={selectedSave}
                onChange={e => { setSelectedSave(e.target.value); setModRows([]); setCheckedMods({}); }}
            >
                {saves?.map(save => (
                    <option key={save.name} value={save.name}>{save.name}</option>
                ))}
            </select>

            <Button
                isDisabled={isDisabled || isLoading}
                isLoading={isLoading}
                onClick={onReadSave}
                className="mr-2"
            >
                {t("mods.load_mods_from_save")}
            </Button>

            {/* Ошибка */}
            {syncError && (
                <div className="mt-4 p-3 bg-red-100 text-red font-bold">
                    ⚠ {syncError}
                </div>
            )}

            {/* Предупреждение */}
            {warning && (
                <div className="mt-4 p-3 bg-yellow-100 text-yellow-800">
                    ⚠ {warning}
                </div>
            )}

            {/* Таблица модов */}
            {modRows.length > 0 && (
                <div className="mt-4">
                    {/* Кнопки выбора */}
                    <div className="flex mb-2 gap-2">
                        <Button size="sm" onClick={selectAll}>{t("mods.sync_from_save")}</Button>
                        <Button size="sm" onClick={clearAll}>{t("cancel")}</Button>
                        <Button
                            size="sm"
                            isDisabled={checkedCount === 0 || isSyncing}
                            isLoading={isSyncing}
                            onClick={onSync}
                        >
                            Sync selected ({checkedCount})
                        </Button>
                    </div>

                    {/* Прогресс */}
                    {currentMod && (
                        <div className="mb-2 text-sm text-orange">
                            <FontAwesomeIcon icon={faSpinner} spin={true} className="mr-2"/>
                            {t("mods.sync_downloading")}: {currentMod}
                        </div>
                    )}

                    <table className="w-full text-sm">
                        <thead>
                            <tr className="border-b font-bold">
                                <td className="py-1 pr-2 w-6"></td>
                                <td className="py-1 pr-4">{t("mods.title")}</td>
                                <td className="py-1 pr-4">{t("mods.mod_list.mod_version")}</td>
                                <td className="py-1 pr-4">{t("mods.status_already_installed")}</td>
                                <td className="py-1">{t("controls.status")}</td>
                            </tr>
                        </thead>
                        <tbody>
                            {/* DLC группа */}
                            {modRows.some(m => DLC_MODS.has(m.name)) && (() => {
                                const dlcMods = modRows.filter(m => DLC_MODS.has(m.name));
                                const dlcKey = 'dlc-group';
                                const allInstalled = dlcMods.every(m => m.status === 'installed' || m.status === 'builtin');
                                return (
                                    <tr key={dlcKey} className="border-b hover:bg-gray-100 cursor-pointer bg-blue-50"
                                        onClick={() => !allInstalled && toggleCheck(dlcKey)}>
                                        <td className="py-1 pr-2"></td>
                                        <td className="py-1 pr-4 italic text-blue-600">
                                            Space Age DLC
                                            <span className="ml-2 text-xs text-gray-500">
                                                (elevated-rails, quality, space-age)
                                            </span>
                                        </td>
                                        <td className="py-1 pr-4">{dlcMods[0]?.version_required}</td>
                                        <td className="py-1 pr-4">{dlcMods[0]?.version_installed || '—'}</td>
                                        <td className="py-1">
                                            <span className="mr-2">{STATUS_ICON[allInstalled ? 'installed' : 'builtin']}</span>
                                            {allInstalled ? t('mods.status_already_installed') : t('mods.status_builtin')}
                                        </td>
                                    </tr>
                                );
                            })()}
                            {/* Остальные моды */}
                            {modRows.filter(m => !DLC_MODS.has(m.name)).map((mod, i) => (
                                <tr
                                    key={i}
                                    className="border-b hover:bg-gray-100 cursor-pointer"
                                    onClick={() => mod.status !== 'builtin' && toggleCheck(mod.name)}
                                >
                                    <td className="py-1 pr-2">
                                        {mod.status !== 'builtin' && mod.status !== 'installed' && (
                                            <input
                                                type="checkbox"
                                                checked={!!checkedMods[mod.name]}
                                                onChange={() => toggleCheck(mod.name)}
                                                onClick={e => e.stopPropagation()}
                                            />
                                        )}
                                    </td>
                                    <td className="py-1 pr-4">
                                        {mod.name}
                                        {mod.status !== 'builtin' && (
                                            <a
                                                href={mod.portal_url}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className="ml-2 text-blue hover:text-blue-light"
                                                onClick={e => e.stopPropagation()}
                                            >
                                                <FontAwesomeIcon icon={faExternalLinkAlt} size="xs"/>
                                            </a>
                                        )}
                                    </td>
                                    <td className="py-1 pr-4">{mod.version_required}</td>
                                    <td className="py-1 pr-4">{mod.version_installed || '—'}</td>
                                    <td className="py-1">
                                        <span className="mr-2">{STATUS_ICON[mod.status]}</span>
                                        {STATUS_TEXT(t)[mod.status] || mod.status}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
};

export default LoadMods;
