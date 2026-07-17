import React, {useState, useEffect} from "react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faTrashAlt, faUpload, faChevronDown, faChevronRight} from "@fortawesome/free-solid-svg-icons";
import presetsResource from "../../../../api/resources/presets";
import ConfirmDialog from "../../../components/ConfirmDialog";
import {useTranslation} from "react-i18next";

const PresetItem = ({preset, serverId, onLoaded, onDeleted}) => {
    const {t} = useTranslation();
    const [isExpanded, setIsExpanded] = useState(false);
    const [detail, setDetail] = useState(null);
    const [isLoadOpen, setIsLoadOpen] = useState(false);
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
    const [isLoading, setIsLoading] = useState(false);

    const toggleExpand = async () => {
        if (!isExpanded && !detail) {
            const d = await presetsResource.get(serverId, preset.name);
            setDetail(d);
        }
        setIsExpanded(v => !v);
    };

    const loadPreset = () => {
        setIsLoading(true);
        presetsResource.load(serverId, preset.name)
            .then(() => {
                window.flash(t("presets.loaded", `Preset "${preset.name}" loaded`), "green");
                if (onLoaded) onLoaded();
            })
            .catch(() => window.flash(t("presets.load_error", "Failed to load preset"), "red"))
            .finally(() => { setIsLoading(false); setIsLoadOpen(false); });
    };

    const deletePreset = () => {
        presetsResource.delete(serverId, preset.name)
            .then(() => {
                window.flash(t("presets.deleted", `Preset "${preset.name}" deleted`), "green");
                if (onDeleted) onDeleted();
            })
            .catch(() => window.flash(t("presets.delete_error", "Failed to delete preset"), "red"));
    };

    return (
        <div className="mb-4">
            <div className="flex items-center justify-between">
                <h2 className="text-lg text-dirty-white mb-1 inline cursor-pointer flex items-center gap-2"
                    onClick={toggleExpand}>
                    <FontAwesomeIcon icon={isExpanded ? faChevronDown : faChevronRight} className="text-xs text-gray-400"/>
                    {preset.name}
                    {preset.description && <span className="text-gray-400 text-sm font-normal">— {preset.description}</span>}
                    <span className="text-gray-500 text-sm font-normal">({preset.mod_count} mods)</span>
                </h2>
                <div className="flex space-x-2 items-center">
                    <FontAwesomeIcon
                        className="text-blue cursor-pointer hover:text-blue-light inline"
                        icon={faUpload}
                        onClick={() => setIsLoadOpen(true)}
                    />
                    <ConfirmDialog
                        title={t("presets.load_preset", "Load Preset")}
                        content={t("presets.load_confirm", `Load preset "${preset.name}"? This will replace current manifest.`).replace("@@@", preset.name)}
                        isOpen={isLoadOpen}
                        close={() => setIsLoadOpen(false)}
                        onSuccess={loadPreset}
                    />
                    {isConfirmingDelete ? (
                        <>
                            <span className="text-red text-sm cursor-pointer font-bold mr-2"
                                  onClick={deletePreset}>{t("servers.confirm_delete", "Confirm")}</span>
                            <span className="text-gray-400 text-sm cursor-pointer"
                                  onClick={() => setIsConfirmingDelete(false)}>✕</span>
                        </>
                    ) : (
                        <FontAwesomeIcon className="text-red cursor-pointer hover:text-red-light inline"
                                         onClick={() => setIsConfirmingDelete(true)} icon={faTrashAlt}/>
                    )}
                </div>
            </div>
            {isExpanded && detail && (
                <div className="mt-1 ml-4">
                    {(detail.mods || []).map(m => (
                        <div key={m.name} className="flex items-center gap-2 text-sm py-0.5">
                            <span className={m.enabled ? "text-green" : "text-gray-500"}>
                                {m.enabled ? "●" : "○"}
                            </span>
                            <span>{m.name}</span>
                            <span className="text-gray-500">{m.version}</span>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

const PresetPanel = ({serverId, presets = [], onLoaded}) => {
    if (presets.length === 0) return null;

    return (
        <div>
            {presets.map(p => (
                <PresetItem
                    key={p.name}
                    preset={p}
                    serverId={serverId}
                    onLoaded={onLoaded}
                    onDeleted={onLoaded}
                />
            ))}
        </div>
    );
};

export default PresetPanel;
