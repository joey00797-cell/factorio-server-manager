import React, {useEffect, useState} from "react";
import savesResource from "../../../api/resources/saves";
import Panel from "../../components/Panel";
import CreateSaveForm from "./components/CreateSaveForm";
import UploadSaveForm from "./components/UploadSaveForm";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {faDownload, faTrashAlt} from "@fortawesome/free-solid-svg-icons";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import serverResource from "../../../api/resources/server";
import ServerScopeHeader from "../../components/ServerScopeHeader";

const Saves = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();

    const [saves, setSaves] = useState([]);
    const [sortField, setSortField] = useState("last_mod");
    const [sortAsc, setSortAsc] = useState(false);
    const [serverStatus, setServerStatus] = useState({running: false});

    const updateList = () => {
        savesResource.list(false, serverId)
            .then(res => {
                if (res) {
                    setSaves(res);
                }
            })

    }

    useEffect(() => {
        updateList();
        serverResource.status(serverId).then(setServerStatus);
    }, [serverId]);

    const deleteSave = async (save) => {
        const res = await savesResource.delete(save, serverId);
        if (res) {
            updateList()
        }
    }

    return (
        <>
            <ServerScopeHeader/>
            <div className="lg:flex mb-6">
                <Panel
                    title={t("saves.create_save")}
                    className="lg:w-1/2 lg:mr-3 mb-6 lg:mb-0"
                    content={
                        serverStatus.running
                            ? <p className="text-red-light pt-4 pb-24">
                                {t("saves.create_new_save_only_when_server_not_running")}
                            </p>
                            : <CreateSaveForm onSuccess={updateList} serverId={serverId}/>
                    }
                />
                <Panel
                    title={t("saves.upload_save")}
                    className="lg:w-1/2 lg:ml-3"
                    content={<UploadSaveForm onSuccess={updateList} serverId={serverId}/>}
                />
            </div>

            <Panel
                className="mb-4"
                title={t("saves.title")}
                content={
                    <div className="overflow-x-auto w-full">
                        <table className="w-full">
                            <thead>
                            <tr className="text-left py-1">
                                {[["name", t("name")], ["last_mod", t("saves.last_modified")], ["size", t("saves.size")]].map(([field, label]) => (
                                    <th key={field} className="cursor-pointer select-none pr-4 hover:text-orange" onClick={() => { if (sortField === field) setSortAsc(a => !a); else { setSortField(field); setSortAsc(true); } }}>
                                        {label}<span className="text-gray-400 text-xs ml-1">{sortField === field ? (sortAsc ? "▲" : "▼") : "↕"}</span>
                                    </th>
                                ))}
                                <th>{t("actions")}</th>
                            </tr>
                            </thead>
                            <tbody>
                            {saves.length === 0 && (
                                <tr>
                                    <td className="py-4 text-gray-light" colSpan="4">
                                        {t("saves.empty", "No saves for this server yet.")}
                                    </td>
                                </tr>
                            )}
                            {saves.slice().sort((a, b) => {
                                let av = sortField === "size" ? a.size : sortField === "last_mod" ? new Date(a.last_mod) : a.name.toLowerCase();
                                let bv = sortField === "size" ? b.size : sortField === "last_mod" ? new Date(b.last_mod) : b.name.toLowerCase();
                                if (av < bv) return sortAsc ? -1 : 1;
                                if (av > bv) return sortAsc ? 1 : -1;
                                return 0;
                            }).map(save =>
                                <tr className="py-2 md:py-1 hover:glow-orange hover:bg-orange hover:text-black cursor-pointer" key={save.name}>
                                    <td className="pr-4">{save.name}</td>
                                    <td className="pr-4">{(new Date(save.last_mod)).toLocaleString()}</td>
                                    <td className="pr-4">{parseFloat(save.size / 1024 / 1024).toFixed(3)} MB</td>
                                    <td>
                                        <a href={savesResource.downloadURL(save.name, serverId)} className="mr-2">
                                            <FontAwesomeIcon
                                                className="text-gray-light cursor-pointer hover:text-orange"
                                                icon={faDownload}/>
                                        </a>
                                        <FontAwesomeIcon className="text-red cursor-pointer hover:text-red-light mr-2"
                                                         onClick={() => deleteSave(save)} icon={faTrashAlt}/>
                                    </td>
                                </tr>
                            )}
                            </tbody>
                        </table>
                    </div>
                }
            />
        </>
    )
}

export default Saves;
