import Panel from "../components/Panel";
import Button from "../components/Button";
import React, {useEffect, useState} from "react";
import modsResource from "../../api/resources/mods";
import {useParams} from "react-router-dom";
import {useTranslation} from "react-i18next";
import ServerScopeHeader from "../components/ServerScopeHeader";

const ModOptions = () => {
    const {serverId} = useParams();
    const {t} = useTranslation();
    const [data, setData] = useState({exists: false, base64: "", size: 0, note: ""});
    const [isSaving, setIsSaving] = useState(false);

    useEffect(() => {
        modsResource.modSettings.get(serverId).then(setData);
    }, [serverId]);

    const save = async () => {
        setIsSaving(true);
        try {
            const saved = await modsResource.modSettings.update(serverId, {base64: data.base64 || ""});
            setData(current => ({...current, exists: true, size: saved.size}));
            window.flash(t("saved"), "green");
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <>
            <ServerScopeHeader/>
            <Panel
                title={t("mods.mod_options", "Mod Options")}
                content={
                    <>
                        <div className="mb-4 text-sm italic">
                            {data.note || t("mods.mod_options_raw_note", "Edit the raw base64 encoded mod-settings.dat for this server.")}
                        </div>
                        <div className="mb-4">
                            <div className="font-bold">{t("saves.size")}</div>
                            <div>{data.size || 0} bytes</div>
                        </div>
                        <textarea
                            className="shadow appearance-none border w-full h-96 py-2 px-3 text-black font-mono"
                            value={data.base64 || ""}
                            onChange={e => setData({...data, base64: e.target.value})}
                        />
                    </>
                }
                actions={<Button type="success" isLoading={isSaving} onClick={save}>{t("save")}</Button>}
            />
        </>
    );
};

export default ModOptions;
