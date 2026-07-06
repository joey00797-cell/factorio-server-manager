import React, {useState} from "react";
import {faSpinner, faTrashAlt, faUpload, faLink} from "@fortawesome/free-solid-svg-icons";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import modsResource from "../../../../api/resources/mods";
import ModList from "./ModList";
import ConfirmDialog from "../../../components/ConfirmDialog";
import { useTranslation } from "react-i18next";

const ModPack = ({modPack, reloadModPacks, factorioVersion, reloadMods, disabled = false}) => {
    const { t } = useTranslation();
    const [isLoading, setIsLoading] = useState(false);
    const [isLoadModPackDialogOpen, setIsLoadModPackDialogOpen] = useState(false);
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);


    const deleteModPack = modName => {
        return modsResource.packs
            .delete(modName)
            .then(reloadModPacks)
    }

    const toggleMod = modName => {
        return modsResource
            .packs
            .mods
            .toggle(modPack.name, modName)
            .then(reloadModPacks)
    }

    const updateMod = version => {
        return modsResource
            .packs
            .mods
            .update(modPack.name, version)
            .then(reloadModPacks)
    }

    const deleteMod = modName => {
        return modsResource
            .packs
            .mods
            .delete(modPack.name, modName)
            .then(reloadModPacks)
    }

    const loadModPack = name => {
        setIsLoading(true)
        return modsResource.packs
            .load(name)
            .then(reloadMods)
            .finally(() => setIsLoading(false))
    }

    return (
        <div className="mb-4">
            <div className="flex items-center justify-between">
                <h2 className="text-lg text-dirty-white mb-1 inline">{modPack.name}</h2>
                <div className="flex space-x-2 items-center">
                    <FontAwesomeIcon
                        className="text-gray-400 cursor-pointer hover:text-orange inline"
                        title={t("mods.mod_pack.copy_link", "Copy download link")}
                        icon={faLink}
                        onClick={() => {
                            const url = `${window.location.origin}/share/${encodeURIComponent(modPack.name)}`;
                            if (navigator.clipboard) {
                                navigator.clipboard.writeText(url).then(() => {
                                    if (window.flash) window.flash(t("mods.mod_pack.link_copied", "Link copied!"), "green");
                                });
                            } else {
                                const el = document.createElement("textarea");
                                el.value = url;
                                document.body.appendChild(el);
                                el.select();
                                document.execCommand("copy");
                                document.body.removeChild(el);
                                if (window.flash) window.flash(t("mods.mod_pack.link_copied", "Link copied!"), "green");
                            }
                        }}
                    />
                    {
                        !disabled &&
                        <>
                            <FontAwesomeIcon className="text-blue cursor-pointer hover:text-blue-light inline"
                                             onClick={() => setIsLoadModPackDialogOpen(true)}
                                             spin={isLoading}
                                             icon={isLoading ? faSpinner : faUpload}
                            />
                            <ConfirmDialog
                                title={t("mods.mod_pack.load_modpack")}
                                content={t("mods.mod_pack.load_confirm_dialog").replace("@@@", modPack.name)}
                                isOpen={isLoadModPackDialogOpen}
                                close={() => setIsLoadModPackDialogOpen(false)}
                                onSuccess={() => loadModPack(modPack.name)}
                            />
                        </>
                    }

                    {isConfirmingDelete ? (
                        <>
                            <span className="text-red text-sm cursor-pointer font-bold mr-2" onClick={() => deleteModPack(modPack.name)}>{t("servers.confirm_delete", "Confirm")}</span>
                            <span className="text-gray-400 text-sm cursor-pointer" onClick={() => setIsConfirmingDelete(false)}>✕</span>
                        </>
                    ) : (
                        <FontAwesomeIcon className="text-red cursor-pointer hover:text-red-light inline"
                                         onClick={() => setIsConfirmingDelete(true)} icon={faTrashAlt}/>
                    )}
                </div>
            </div>
            <ModList mods={modPack.mods.mods}
                     factorioVersion={factorioVersion}
                     toggleMod={toggleMod}
                     updateMod={updateMod}
                     deleteMod={deleteMod}
            />
        </div>
    )
}

export default ModPack;