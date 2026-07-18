import Panel from "../../components/Panel";
import React, {useEffect, useState} from "react";
import modsResource from "../../../api/resources/mods";
import Button from "../../components/Button";
import server from "../../../api/resources/server";
import TabControl from "../../components/Tabs/TabControl";
import Tab from "../../components/Tabs/Tab";
import Fuse from "fuse.js";
import CreateModPack from "./components/CreateModPack";
import CreatePreset from "./components/CreatePreset";
import ModPack from "./components/ModPack";
import ModList from "./components/ModList";
import ModOptionsTab from "./components/ModOptionsTab";
import ModLibrary from "./components/ModLibrary";
import PresetPanel from "./components/PresetPanel";
import presetsResource from "../../../api/resources/presets";
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
    const [presetList, setPresetList] = useState([]);
    const [preview, setPreview] = useState(null);

    const addUpdatableMod = mod => {
        setUpdatableMods(mods => [...mods, mod])
    };

    const fetchManifest = () => {
        modLibrary.manifest.get(serverId).then(m => setManifest(m || {items: []})).catch(() => {});
    };

    const fetchLibraryMods = () => {
        modLibrary.list().then(assets => setLibraryMods(assets || [])).catch(() => {});
    }
    const fetchPresets = () => {
        presetsResource.list(serverId).then(data => setPresetList(data || [])).catch(() => {});
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
        fetchPresets();
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

    const deleteMod = async (modName) => {
        const asset = libraryMods.find(m => m.name === modName);
        if (!asset) return;
        const allItems = (manifest.items || []);
        console.log('[deleteMod] modName:', modName, 'asset.id:', asset.id, 'typeof:', typeof asset.id);
        console.log('[deleteMod] allItems:', JSON.stringify(allItems.map(i => ({asset_id: i.asset_id, enabled: i.enabled, type: typeof i.asset_id}))));
        const currentItems = allItems.map(i => ({
            asset_id: i.asset_id,
            enabled: i.enabled,
            to_delete: i.to_delete || false
        }));
        console.log('[deleteMod] asset:', asset.id, typeof asset.id);
        console.log('[deleteMod] allItems asset_ids:', allItems.map(i => ({id: i.asset_id, type: typeof i.asset_id, enabled: i.enabled})));
        const alreadyInManifest = allItems.find(i => i.asset_id === asset.id);
        console.log('[deleteMod] alreadyInManifest:', alreadyInManifest);

        if (alreadyInManifest) {
            // Mark as to_delete in manifest -> Apply will delete from library
            const newItems = currentItems.map(i => {
                if (i.asset_id === asset.id) {
                    return {...i, to_delete: true};
                }
                return i;
            });
            const updated = await modLibrary.manifest.update(serverId, newItems);
            setManifest(updated);
        } else {
            // Not in manifest -> add with to_delete so Apply removes from library
            const newItems = [...currentItems, {asset_id: asset.id, enabled: false, to_delete: true}];
            const updated = await modLibrary.manifest.update(serverId, newItems);
            setManifest(updated);
        }
        fetchLibraryMods();
    }

    const updateMod = version => {
        return modsResource
            .update(version, serverId)
            .then(fetchInstalledMods)
    }

    useEffect(() => { if (serverId) loadPreview(); }, [manifest, libraryMods]);

    let disabled = serverStatus.running

    const manifestAssetIds = new Set((manifest.items || []).filter(i => !i.to_delete).map(i => i.asset && i.asset.name).filter(Boolean));

    const onAddToPreset = async (mod, presetName) => {
        try {
            await presetsResource.addMod(serverId, presetName, mod.id);
            fetchPresets();
            window.flash(`${mod.name} added to preset "${presetName}"`, "green");
        } catch (e) {
            window.flash("Failed to add mod to preset", "red");
        }
    };

    const onDLCToggle = async (dlcNames, enable) => {
        console.log('[DLC toggle] names:', dlcNames, 'enable:', enable, 'libraryMods DLC:', libraryMods.filter(m => dlcNames.includes(m.name)));
        const allItems = (manifest.items || []);
        const currentItems = allItems.map(i => ({
            asset_id: i.asset_id,
            enabled: i.enabled,
            to_delete: i.to_delete || false
        }));
        let newItems = [...currentItems];
        for (const name of dlcNames) {
            const asset = libraryMods.find(m => m.name === name);
            if (!asset) continue;
            const existing = newItems.find(i => i.asset_id === asset.id);
            if (existing) {
                existing.enabled = enable;
            } else {
                newItems.push({asset_id: asset.id, enabled: enable, to_delete: false});
            }
        }
        const updated = await modLibrary.manifest.update(serverId, newItems);
        setManifest(updated);
    };

    const onManifestToggle = async (mod) => {
        console.log('[toggle] mod:', mod.name, 'id:', mod.id, 'inManifest check:', manifestAssetIds.has(mod.name));
        console.log('[toggle] manifestAssetIds:', [...manifestAssetIds]);
        console.log('[toggle] manifest.items:', JSON.stringify((manifest.items || []).map(i => ({asset_id: i.asset_id, name: i.asset?.name, enabled: i.enabled}))));
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
                        <ModLibrary serverId={serverId} onModUploaded={fetchLibraryMods} preview={preview} libraryMods={libraryMods} onApplied={(result) => { setPreview(result); fetchLibraryMods(); fetchManifest(); }}/>
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
                             mods={libraryMods.map(m => {
                                 const manifestItem = (manifest.items || []).find(i => i.asset_id === m.id);
                                 return manifestItem ? {...m, enabled: manifestItem.enabled && !manifestItem.to_delete, to_delete: manifestItem.to_delete || false} : m;
                             })}
                             factorioVersion={factorioVersion}
                             disabled={disabled}
                             manifestAssetIds={manifestAssetIds}
                             onManifestToggle={onManifestToggle}
                             onDLCToggle={onDLCToggle}
                             presets={presetList}
                             onAddToPreset={onAddToPreset}
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
                        <span className="ml-2"><CreatePreset serverId={serverId} onSuccess={fetchPresets}/></span>
                        <span className="ml-2"><CreateModPack onSuccess={fetchModPacks}/></span>
                    </>
                }
            />

            <div className={`grid grid-cols-1 gap-4 mb-6${presetList.length > 0 && modPacks.length > 0 ? " md:grid-cols-2" : ""}`}>
                {presetList.length > 0 && (
                    <div>
                        <Panel
                            title={t("presets.title", "Mod Presets")}
                            content={
                                <PresetPanel serverId={serverId} presets={presetList} onLoaded={() => { fetchManifest(); fetchLibraryMods(); fetchPresets(); }}/>
                            }
                        />
                    </div>
                )}
                {modPacks.length > 0 && (
                    <div>
                        <Panel
                            title={t("mods.mod_packs")}
                            content={modPacks.map(pack =>
                                <ModPack factorioVersion={factorioVersion}
                                         key={pack.name}
                                         modPack={pack}
                                         reloadMods={fetchInstalledMods}
                                         reloadModPacks={fetchModPacks}
                                         disabled={disabled}
                                />
                            )}

                        />
                    </div>
                )}
            </div>

            </>}
        </div>
    )
}

export default Mods;
