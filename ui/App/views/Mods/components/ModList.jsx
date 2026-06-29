import Mod from "./Mod";
import React from "react";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faCheck, faTimes, faToggleOff, faToggleOn} from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";

const DLC_MODS = new Set(['elevated-rails', 'quality', 'space-age']);

const ModList = ({mods, factorioVersion, updateMod, toggleMod, deleteMod, addUpdatableMod = null, disabled = false}) => {
    const { t } = useTranslation();

    const dlcMods = mods.filter(m => DLC_MODS.has(m.name));
    const regularMods = mods.filter(m => !DLC_MODS.has(m.name));
    const dlcEnabled = dlcMods.some(m => m.enabled);
    const DLC_LIST = ["elevated-rails", "quality", "space-age"];
    const toggleDLC = () => {
        if (dlcMods.length > 0) {
            dlcMods.forEach(m => {
                if (dlcEnabled === m.enabled) {
                    toggleMod(m.name);
                }
            });
        } else {
            DLC_LIST.forEach(name => toggleMod(name));
        }
    };

    return (
        <table className="w-full">
            <thead>
                <tr className="text-left py-1">
                    <th>{t("name")}</th>
                    <th>{t("mods.mod_list.enabled")}</th>
                    <th>{t("mods.mod_list.compatibility")}</th>
                    <th>{t("mods.mod_list.mod_version")}</th>
                    <th>{t("mods.mod_list.factorio_version")}</th>
                    <th/>
                </tr>
            </thead>
            <tbody>
                {/* DLC группа */}
                {factorioVersion !== null && (
                    <tr className="py-1 bg-blue-50 hover:glow-orange hover:bg-orange hover:text-black">
                        <td className="pr-4 italic text-blue-600">
                            Space Age DLC
                            <span className="ml-2 text-xs text-gray-500 not-italic">
                                (elevated-rails, quality, space-age)
                            </span>
                        </td>
                        <td className="pr-4">
                            {disabled
                                ? dlcEnabled
                                    ? <FontAwesomeIcon className="text-green" icon={faCheck}/>
                                    : <FontAwesomeIcon className="text-red" icon={faTimes}/>
                                : dlcEnabled
                                    ? <FontAwesomeIcon className="cursor-pointer hover:text-green-light text-green"
                                                       icon={faToggleOn} onClick={toggleDLC}/>
                                    : <FontAwesomeIcon className="cursor-pointer hover:text-red-light text-red"
                                                       icon={faToggleOff} onClick={toggleDLC}/>
                            }
                        </td>
                        <td className="pr-4">
                            <FontAwesomeIcon className="text-green" icon={faCheck}/>
                        </td>
                        <td className="pr-4">{dlcMods[0]?.version}</td>
                        <td className="pr-4">{dlcMods[0]?.factorio_version}</td>
                        <td/>
                    </tr>
                )}
                {/* Остальные моды */}
                {factorioVersion !== null && regularMods.map(
                    (mod, i) =>
                        <Mod mod={mod} key={i}
                             updateMod={updateMod}
                             toggleMod={toggleMod}
                             deleteMod={deleteMod}
                             addUpdatableMod={addUpdatableMod}
                             factorioVersion={factorioVersion}
                             disabled={disabled}
                        />
                )}
            </tbody>
        </table>
    );
};

export default ModList;
