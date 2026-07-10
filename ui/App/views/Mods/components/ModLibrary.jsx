import React, {useEffect, useState} from "react";
import Panel from "../../../components/Panel";
import Button from "../../../components/Button";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faExternalLinkAlt, faSpinner} from "@fortawesome/free-solid-svg-icons";
import modLibrary from "../../../../api/resources/modLibrary";
import modsResource from "../../../../api/resources/mods";
import {useTranslation} from "react-i18next";
import SelectVersionForm from "./AddMod/components/SelectVersionForm";
import FactorioLogin from "./AddMod/components/FactorioLogin";
import Fuse from "fuse.js";

const ModLibrary = ({serverId, onModUploaded, onApplied, preview, onPreviewLoad}) => {
    const {t} = useTranslation();
    const [isApplying, setIsApplying] = useState(false);
    const [uploadFile, setUploadFile] = useState(null);
    const [isUploading, setIsUploading] = useState(false);

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
    }, []);

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

    const applyManifest = async () => {
        setIsApplying(true);
        try {
            const result = await modLibrary.manifest.apply(serverId);
            window.flash(t("mods.apply_success", "Mods applied successfully"), "green");
            if (onApplied) onApplied(result);
        } catch (e) {
            window.flash(t("mods.apply_error", "Failed to apply mods"), "red");
        } finally {
            setIsApplying(false);
        }
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

    return (
        <>
            <Panel
                title={t("mods.mod_library", "Mod Library")}
                className="mb-4"
                content={
                    <div className="space-y-4">
                        {/* Upload ZIP */}
                        <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-bold text-sm">{t("mods.upload_zip", "Upload ZIP:")}</span>
                            <input type="file" accept=".zip" className="text-sm" onChange={e => setUploadFile(e.target.files[0])}/>
                            <Button size="sm" onClick={uploadMod} isLoading={isUploading} isDisabled={!uploadFile}>
                                {t("mods.upload_to_library", "Upload to Library")}
                            </Button>
                        </div>

                        {/* Portal */}
                        <div>
                            <div className="font-bold text-sm mb-2">
                                {t("mods.mod_portal", "Mod Portal")}
                                <a href="https://mods.factorio.com" target="_blank" rel="noopener noreferrer" className="ml-2 text-blue hover:text-blue-light text-xs">
                                    <FontAwesomeIcon icon={faExternalLinkAlt}/>
                                </a>
                            </div>
                            {!isPortalAuth ? (
                                <FactorioLogin setIsFactorioAuthenticated={setIsPortalAuth} setPortalUsername={setPortalUsername}/>
                            ) : (
                                <div>
                                    <div className="flex items-center gap-2 mb-2">
                                        <span className="text-sm text-green font-bold">{portalUsername}</span>
                                        <Button size="sm" type="danger" onClick={logout}>{t("logout")}</Button>
                                    </div>
                                    <div className="relative">
                                        <input
                                            type="text"
                                            className="shadow border w-full py-1 px-2 text-black"
                                            placeholder={t("mods.search_portal", "Search mod portal...")}
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
                                    {selectedMod && (
                                        <Button size="sm" className="mt-2" onClick={openVersionModal} isLoading={isImporting}>
                                            {t("mods.import_to_library", "Import to Library")}
                                        </Button>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>
                }
            />

            {/* Preview */}
            {preview && ((preview.to_add?.length || 0) + (preview.to_remove?.length || 0) + (preview.to_enable?.length || 0) + (preview.to_disable?.length || 0) > 0) && (
                <Panel
                    title={t("mods.apply_preview", "Apply Preview")}
                    className="mb-4"
                    content={
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
                                    {(preview.to_add || []).map(m => <div key={m.name}>{m.name} {m.version}</div>)}
                                </div>
                                <div>
                                    <div className="font-bold text-red mb-1">{t("mods.to_remove", "To Remove")} ({preview.to_remove?.length || 0})</div>
                                    {(preview.to_remove || []).map(m => <div key={m.name}>{m.name} {m.version}</div>)}
                                </div>
                            </div>
                            <Button size="sm" type="success" onClick={applyManifest} isLoading={isApplying} isDisabled={!preview.can_apply}>
                                {t("mods.apply_to_server", "Apply to Server")}
                            </Button>
                        </div>
                    }
                />
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
