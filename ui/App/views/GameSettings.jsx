import Panel from "../components/Panel";
import Button from "../components/Button";
import React, {useEffect, useState} from "react";
import settingsResource from "../../api/resources/settings";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import ServerScopeHeader from "../components/ServerScopeHeader";

const GameSettings = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();

    const [settingsCategories, setSettingsCategories] = useState({});
    const [loadError, setLoadError] = useState(false);
    const [isSaving, setIsSaving] = useState(false);

    const fetchSettings = async () => {
        try {
            const res = await settingsResource.game.list(serverId);
            setSettingsCategories(res || {});
            setLoadError(false);
        } catch (e) {
            setLoadError(true);
        }
    };

    useEffect(() => {
        fetchSettings();
    }, [serverId]);

    const updateValue = (section, key, value) => {
        setSettingsCategories(current => ({
            ...current,
            [section]: {
                ...(current[section] || {}),
                [key]: value,
            }
        }));
    };

    const save = async () => {
        setIsSaving(true);
        try {
            await settingsResource.game.update(settingsCategories, serverId);
            window.flash(t("saved"), "green");
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <>
            <ServerScopeHeader/>
            {loadError && (
                <div className="mb-4 p-3 bg-red bg-opacity-20 border border-red rounded text-red-light font-bold">
                    {t("game_settings.not_available")}
                </div>
            )}
            <Panel
                className="mb-4"
                title={t("game_settings.title")}
                content={
                    <>
                        {Object.keys(settingsCategories).map(section => {
                            const settings = settingsCategories[section] || {};
                            return (
                                <div key={section} className="mb-6">
                                    <h1 className="mb-2 text-lg text-dirty-white">{section}</h1>
                                    <div className="grid gap-3">
                                        {Object.keys(settings).map(key => (
                                            <label className="block" key={`${section}-${key}`}>
                                                <span className="block font-bold mb-1">{key}</span>
                                                <input
                                                    className="shadow appearance-none border w-full py-2 px-3 text-black"
                                                    value={settings[key]}
                                                    onChange={e => updateValue(section, key, e.target.value)}
                                                />
                                            </label>
                                        ))}
                                    </div>
                                </div>
                            );
                        })}
                    </>
                }
                actions={<Button type="success" isLoading={isSaving} onClick={save}>{t("save")}</Button>}
            />
        </>
    );
};

export default GameSettings;
