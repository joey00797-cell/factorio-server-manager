import React, {useEffect, useState} from "react";
import serverResource from "../../../../../../api/resources/server";
import Modal from "../../../../../components/Modal";
import Button from "../../../../../components/Button";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faCloudDownloadAlt} from "@fortawesome/free-solid-svg-icons/faCloudDownloadAlt";
import {faCheck} from "@fortawesome/free-solid-svg-icons/faCheck";
import {faTimes} from "@fortawesome/free-solid-svg-icons/faTimes";
import { useTranslation } from "react-i18next";

const SelectVersionForm = ({releases, isOpen, close, install, serverId}) => {
    const { t } = useTranslation();
    const [serverVer, setServerVer] = useState("");

    useEffect(() => {
        if (!serverId) return;
        serverResource.status(serverId).then(d => setServerVer((d && d.base_mod_version) || "")).catch(() => {});
    }, [serverId]);

    useEffect(() => {
        const onKey = e => { if (e.key === "Escape" && close) close(); };
        window.addEventListener("keydown", onKey);
        return () => window.removeEventListener("keydown", onKey);
    }, [close]);

    // returns {ok, reason}; unknown data counts as compatible
    const check = release => {
        const need = (release.info_json && release.info_json.factorio_version) || release.factorio_version || "";
        if (!need || !serverVer) return {ok: true, reason: ""};
        const mm = need.split(".").slice(0, 2).join(".");
        const ok = serverVer === mm || serverVer.startsWith(mm + ".");
        return {ok, reason: ok ? "" : "needs Factorio " + mm + ", server has " + serverVer};
    };

    const download = release => {
        install(release)
        close()
    }

    return (
        <Modal
            isOpen={isOpen}
            title={t("mods.add_mod.version")}
            content={
                <div className="h-64 overflow-y-auto">
                    <table className="w-full">
                        <thead>
                        <tr className="text-left py-1">
                            <th>Version</th>
                            <th>Compatibility</th>
                            <th>Actions</th>
                        </tr>
                        </thead>
                        <tbody>
                        {[...releases].reverse().map((release, i) =>
                            <tr className="py-2 md:py-1" key={i}>
                                <td className="pr-4">{release.version}</td>
                                <td className="pr-4">{check(release).ok
                                    ? <FontAwesomeIcon  className="text-green" icon={faCheck}/>
                                    : <><FontAwesomeIcon  className="text-red" icon={faTimes}/> <span className="text-xs text-red">{check(release).reason}</span></>
                                }</td>
                                <td>
                                    { check(release).ok && <FontAwesomeIcon className="cursor-pointer" onClick={() => download(release)} icon={faCloudDownloadAlt}/> }
                                </td>
                            </tr>
                        )}
                        </tbody>
                    </table>
                </div>
            }
            actions={
                <Button onClick={close} size="sm" type="danger">Cancel</Button>
            }
        />
    )
}

export default SelectVersionForm;