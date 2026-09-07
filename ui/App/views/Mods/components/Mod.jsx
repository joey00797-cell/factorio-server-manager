import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {
    faArrowCircleUp,
    faExternalLinkAlt,
    faCheck,
    faSpinner,
    faTimes,
    faToggleOff,
    faToggleOn,
    faTrashAlt,
    faBookmark
} from "@fortawesome/free-solid-svg-icons";
import modsResource from "../../../../api/resources/mods";
import React, {useEffect, useState} from "react";
import { useTranslation } from "react-i18next";
import {coerce, gt, satisfies} from "semver";

const Mod = ({mod, factorioVersion, toggleMod, deleteMod, updateMod, addUpdatableMod, disabled = false, inManifest = null, onManifestToggle = null, onVersionSwitch = null, presets = [], onAddToPreset = null, allVersions = []}) => {
    const versions = allVersions || [];
    const { t } = useTranslation();

    const [newVersion, setNewVersion] = useState(null)
    const [showPresetDropdown, setShowPresetDropdown] = useState(false)
    const [showVersionDropdown, setShowVersionDropdown] = useState(false)
    const [icon, setIcon] = useState(faArrowCircleUp)

    useEffect(() => {
        if (!showVersionDropdown) return;
        const handler = () => setShowVersionDropdown(false);
        document.addEventListener('click', handler);
        return () => document.removeEventListener('click', handler);
    }, [showVersionDropdown]);
    const portalUrl = `https://mods.factorio.com/mod/${encodeURIComponent(mod.name)}`

    useEffect(() => {
        if (!disabled) {
            (async () => {
                let data;
                try {
                    data = await modsResource.portal.info(mod.name);
                } catch (e) {
                    // Mod not found on portal (local/custom mod) - skip update check
                    return;
                }
                if (!data || !data.releases) return;

                //get newest COMPATIBLE release
                let newestRelease;
                data.releases.forEach(release => {
                    if (
                        gt(
                            coerce(release.version),
                            coerce(mod.version)
                        ) && (
                            satisfies(factorioVersion, "~" + coerce(release.info_json.factorio_version).version) ||
                            (
                                satisfies(factorioVersion, "1.0.0") &&
                                satisfies(coerce(release.info_json.factorio_version), "0.18.x")
                            )
                        )
                    ) {
                        if (!newestRelease) {
                            newestRelease = release;
                        } else if (gt(coerce(release.version).version, coerce(newestRelease.version).version)) {
                            newestRelease = release;
                        }
                    }
                });

                if (newestRelease && newestRelease.version !== mod.version) {
                    const installableVersion = {
                        downloadUrl: newestRelease.download_url,
                        fileName: newestRelease.file_name,
                        modName: mod.name,
                        version: newestRelease.version
                    }
                    setNewVersion(installableVersion);
                    if (addUpdatableMod !== null) {
                        addUpdatableMod(installableVersion)
                    }
                } else {
                    setNewVersion(null);
                }

            })();
        }
    }, [mod]);

    const isCompatible = () => {
        if (!factorioVersion || !mod.factorio_version) return true;
        const serverMajor = parseInt(factorioVersion.split(".")[0]);
        const modMajor = parseInt(mod.factorio_version.split(".")[0]);
        return serverMajor === modMajor;
    };
    const compatible = isCompatible();

    return (
        <tr className={`py-1 ${mod.to_delete ? "bg-red-900 opacity-60" : !compatible ? "bg-red-950 opacity-70 hover:glow-orange hover:bg-orange hover:text-black" : "hover:glow-orange hover:bg-orange hover:text-black"}`}>
            <td className="pr-4">
                <span className={mod.to_delete ? "line-through text-red-400" : ""}>{mod.title}</span>
                {mod.name && (
                    <a
                        href={portalUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="ml-2 text-blue hover:text-blue-light"
                    >
                        <FontAwesomeIcon icon={faExternalLinkAlt} size="xs"/>
                    </a>
                )}
            </td>
            <td className="pr-4">
                {onManifestToggle !== null
                    ? (inManifest
                        ? <FontAwesomeIcon className="cursor-pointer hover:text-green-light text-green" icon={faToggleOn} onClick={() => compatible && onManifestToggle(mod)} title={compatible ? "" : "Incompatible with server version"}/>
                        : <FontAwesomeIcon className={`cursor-pointer ${compatible ? "hover:text-gray text-gray-500" : "text-red opacity-40 cursor-not-allowed"}`} icon={faToggleOff} onClick={() => compatible && onManifestToggle(mod)} title={compatible ? "" : "Incompatible with server version"}/>)
                    : (mod.enabled
                        ? <FontAwesomeIcon className="text-green" icon={faCheck}/>
                        : <FontAwesomeIcon className="text-red" icon={faTimes}/>)
                }
            </td>
            <td className="pr-4">
                <span className="relative inline-block">
                    {versions.length > 1 ? (
                        <>
                            <span className="cursor-pointer hover:text-orange"
                                onClick={e => { e.stopPropagation(); setShowVersionDropdown(v => !v); }}>
                                {mod.version} ▾
                            </span>
                            {showVersionDropdown && (
                                <div className="absolute left-0 bottom-6 bg-white border border-gray-300 z-50 shadow-lg" style={{minWidth: "130px"}}>
                                    {versions.map(v => {
                                        const compat = !factorioVersion || !v.factorio_version ||
                                            parseInt(factorioVersion.split(".")[0]) === parseInt(v.factorio_version.split(".")[0]);
                                        return (
                                            <div key={v.id}
                                                className={`px-2 py-1 text-xs ${compat ? "cursor-pointer hover:bg-orange text-black" : "text-gray-400 cursor-not-allowed"} ${v.id === mod.id ? "font-bold" : ""}`}
                                                onClick={() => { if (compat && onVersionSwitch) { onVersionSwitch(v); setShowVersionDropdown(false); } }}>
                                                {v.version} {!compat ? "(incompatible)" : ""}
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </>
                    ) : (
                        <>{mod.version}</>
                    )}
                    {!disabled && newVersion && !versions.some(v => v.version === newVersion.version || v.version === newVersion.version.replace(/\.0+$/, '') || newVersion.version === v.version + '.0') && (
                        <FontAwesomeIcon spin={icon === faSpinner}
                            onClick={() => { setIcon(faSpinner); updateMod(newVersion).finally(() => setIcon(faArrowCircleUp)); }}
                            className="hover:text-orange cursor-pointer ml-1"
                            icon={icon}/>
                    )}
                </span>
            </td>
            <td className="pr-4">{mod.factorio_version}</td>

            {
                !disabled &&
                <td className="pr-4">
                    <FontAwesomeIcon className={"text-red cursor-pointer hover:text-red-light"}
                                     onClick={() => deleteMod(mod.name)} icon={faTrashAlt}/>
                </td>
            }
        </tr>
    )
}

export default Mod;

