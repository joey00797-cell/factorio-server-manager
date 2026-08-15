import Button from "../../../components/Button";
import React, {useEffect, useState} from "react";
import modsResource from "../../../../api/resources/mods";
import {useTranslation} from "react-i18next";

const ModOptionsTab = ({serverId}) => {
    const {t} = useTranslation();
    const [data, setData] = useState({exists: false, base64: "", size: 0, note: ""});
    const [isSaving, setIsSaving] = useState(false);

    const handleUpload = (e) => {
        const file = e.target.files[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = () => {
            const b64 = reader.result.split(",")[1];
            setData(current => ({...current, base64: b64, size: file.size}));
            window.flash(t("mods.mod_options_file_loaded", "File loaded, click Save to apply"), "green");
        };
        reader.readAsDataURL(file);
    };

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
        <div>
            <div className="mb-4 text-sm italic">
                {data.note || t("mods.mod_options_raw_note", "Edit the raw base64 encoded mod-settings.dat for this server.")}
            </div>
            <div className="mb-4">
                <div className="font-bold">{t("saves.size")}</div>
                <div>{data.size || 0} bytes</div>
            </div>
            <textarea
                className="shadow appearance-none border w-full h-96 py-2 px-3 text-black font-mono mb-4"
                value={data.base64 || ""}
                onChange={e => setData({...data, base64: e.target.value})}
            />
            <div className="flex gap-2 items-center">
                <Button type="success" isLoading={isSaving} onClick={save}>{t("save")}</Button>
                <label className="cursor-pointer">
                    <Button type="default" onClick={() => document.getElementById('mod-settings-upload').click()}>
                        {t("upload") || "Upload"}
                    </Button>
                    <input id="mod-settings-upload" type="file" accept=".dat" className="hidden" onChange={handleUpload}/>
                </label>
            </div>
        </div>
    );
};

export default ModOptionsTab;
