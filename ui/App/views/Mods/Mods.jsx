import Panel from "../../components/Panel";
import React, {useEffect, useState} from "react";
import modsResource from "../../../api/resources/mods";
import Button from "../../components/Button";
import server from "../../../api/resources/server";
import TabControl from "../../components/Tabs/TabControl";
import Tab from "../../components/Tabs/Tab";
import Fuse from "fuse.js";
import CreateModPack from "./components/CreateModPack";
import ModPack from "./components/ModPack";
import ModList from "./components/ModList";
import ModOptionsTab from "./components/ModOptionsTab";
import ModLibrary from "./components/ModLibrary";
import modLibrary from "../../../api/resources/modLibrary";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import ServerScopeHeader from "../../components/ServerScopeHeader";

const Mods = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();
    const [installedMods, setInstalledMods] = useState([]);
    const [modPacks, setModPacks] = useState([])
    const [factorioVersion, setFactorioVersion] = useState(null);
    const [fuse, setFuse] = useState(undefined);
    const [isDeletingAllMods, setIsDeletingAllMods] = useState(false);
    const [isUpdatingAllMods, setIsUpdatingAllMods] = useState(false);
    const [updatableMods, setUpdatableMods] = useState([]);
    const [serverStatus, setServerStatus] = useState({running: false});
    const [activeTab, setActiveTab] = useState(0);
    const [manifest, setManifest] = useState({items: []});
    const [libraryMods, setLibraryMods] = useState([]);
    const [preview, setPreview] = useState(null);

    const addUpdatableMod = mod => {
        setUpdatableMods(mods => [...mods, mod])
    };

    const fetchManifest = () => {
        modLibrary.manifest.get(serverId).then(m => setManifest(m || {items: []})).catch(() => {});
    };

    const fetchLibraryMods = () => {
        modLibrary.list().then(assets => setLibraryMods(assets || [])).catch(() => {});
    };

    const loadPreview = () => {
        modLibrary.manifest.preview(serverId).then(setPreview).catch(() => {});
    };

    const fetchInstalledMods = () => {
        modsResource.installed(serverId)
            .then(setInstalledMods);
    };

    const fetchModPacks = () => {
        modsResource.packs.list()
            .then(setModPacks)
    }

    const deleteAllMods = () => {
        setIsDeletingAllMods(true);
        modsResource.deleteAll(serverId)
            .then(fetchInstalledMods)
            .finally(() => setIsDeletingAllMods(false))
    }

    const updateAllMods = () => {
        setIsUpdatingAllMods(true);

        let promises = [];
        for (const updatableMod of updatableMods) {
            promises.push(modsResource.update(updatableMod, serverId))
        }

        Promise.all(promises)
            .then(fetchInstalledMods)
            .finally(() => setIsUpdatingAllMods(false));
    }

    useEffect(() => {
        server.status(serverId)
            .then(data => {
                setServerStatus(data);
                setFactorioVersion(data.base_mod_version)
                fetchInstalledMods();
                fetchModPacks();
            })

        fetchManifest();
        fetchLibraryMods();
        // fetch list of mods
        modsResource.portal.list()
            .then(res => {
                setFuse(new Fuse(res.results, {
                    keys: [
                        {
                            "name": "name",
                            weight: 2
                        },
                        {
                            "name": "title",
                            weight: 1
                        }
                    ],
                    minMatchCharLength: 3
                }));
            });

    }, [serverId]);

    const toggleMod = modName => {
        return modsResource
            .toggle(modName, serverId)
            .then(fetchInstalledMods)
    }

    const deleteMod = modName => {
        return modsResource
            .delete(modName, serverId)
            .then(fetchInstalledMods)
    }

    const updateMod = version => {
        return modsResource
            .update(version, serverId)
            .then(fetchInstalledMods)
    }

    useEffect(() => { if (serverId) loadPreview(); }, [manifest, libraryMods]);

    let disabled = serverStatus.running

    const manifestAssetIds = new Set((manifest.items || []).map(i => i.asset && i.asset.name).filter(Boolean));

    const onManifestToggle = async (mod) => {
        const inManifest = manifestAssetIds.has(mod.name);
        const currentItems = (manifest.items || []).map(i => ({asset_id: i.asset_id, enabled: i.enabled}));
        let newItems;
        if (inManifest) {
            const asset = (manifest.items || []).find(i => i.asset && i.asset.name === mod.name);
            newItems = currentItems.filter(i => i.asset_id !== (asset && asset.asset_id));
        } else {
            newItems = [...currentItems, {asset_id: mod.id, enabled: true}];
        }
        const updated = await modLibrary.manifest.update(serverId, newItems);
        setManifest(updated);
    };

    return (
        <div>
            <ServerScopeHeader/>
            {disabled ?
                <Panel className="mb-6"
                       content={
                           <div className="text-red font-bold text-xl">
                               {t("mods.change_mods_while_running_error_message")}
                           </div>
                       }
                />
                :
                <TabControl onChange={setActiveTab}>
                    <Tab title={t("mods.mod_library", "Mod Library")}>
                        <ModLibrary serverId={serverId} onModUploaded={fetchLibraryMods} preview={preview} onApplied={(result) => { setPreview(result); fetchLibraryMods(); fetchManifest(); }}/>
                    </Tab>
                    <Tab title={t("mods.mod_options", "Mod Options")}>
                        <ModOptionsTab serverId={serverId}/>
                    </Tab>
                </TabControl>
            }
            {!disabled && activeTab === 3 ? null : <>
            <Panel
                title={t("mods.title")}
                className="mb-6"
                content={
                    <ModList addUpdatableMod={addUpdatableMod}
                             toggleMod={toggleMod}
                             updateMod={updateMod}
                             deleteMod={deleteMod}
                             mods={libraryMods}
                             factorioVersion={factorioVersion}
                             disabled={disabled}
                             manifestAssetIds={manifestAssetIds}
                             onManifestToggle={onManifestToggle}
                    />
                }
                actions={
                    <>
                        {
                            !disabled &&
                            <Button size="sm" className="mr-2" type="danger" isLoading={isDeletingAllMods}
                                    onClick={deleteAllMods}>{t("mods.delete_all")}</Button> &&
                            <Button size="sm" className="mr-2" isLoading={isUpdatingAllMods}
                                    onClick={updateAllMods}>{t("mods.update_all")}</Button>
                        }
                        <a className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold"
                           href={modsResource.downloadAllURL(serverId)}>{t("mods.download_all")}</a>
                    </>
                }
            />

            <Panel
                title={t("mods.mod_packs")}
                className="mb-6"
                content={
                    modPacks.length === 0 ? (
                        <div className="text-gray-400 text-sm italic">{t("mods.no_mod_packs", "No mod packs yet. Create one to save your current mod setup.")}</div>
                    ) : modPacks.map(
                        (pack) =>
                            <ModPack factorioVersion={factorioVersion}
                                     key={pack.name}
                                     modPack={pack}
                                     reloadMods={fetchInstalledMods}
                                     reloadModPacks={fetchModPacks}
                                     disabled={disabled}
                            />
                    )
                }
                actions={<CreateModPack onSuccess={fetchModPacks}/>}
            />
            </>}
        </div>
    )
}

export default Mods;
