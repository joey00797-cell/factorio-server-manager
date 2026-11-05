import React, {useEffect, useState} from "react";
import {useTranslation} from "react-i18next";
import Modal from "./Modal";
import Button from "./Button";
import appResource from "../../api/resources/app";

const DOCKER_CMD = "docker pull ghcr.io/joey00797-cell/fsm:latest";
const LINUX_CMD = "curl -fsSLO https://raw.githubusercontent.com/joey00797-cell/factorio-server-manager/master/install.sh && FSM_DIR=$PWD bash install.sh";

const renderNotes = (text) => text.split("\n").map((raw, i) => {
    const line = raw.replace(/`/g, "").trimEnd();
    if (!line.trim()) return <div key={i} className="h-2"/>;
    const heading = line.match(/^#{1,6}\s+(.*)$/);
    if (heading) return <div key={i} className="font-bold mt-2">{heading[1]}</div>;
    const item = line.match(/^\s*[-*]\s+(.*)$/);
    if (item) return <div key={i} className="pl-3">• {item[1]}</div>;
    return <div key={i}>{line}</div>;
});

const Command = ({children}) => (
    <pre className="bg-black text-white p-2 text-xs overflow-x-auto whitespace-pre-wrap mb-2">{children}</pre>
);

const VersionBadge = () => {
    const {t} = useTranslation();
    const [info, setInfo] = useState(null);
    const [isOpen, setIsOpen] = useState(false);

    useEffect(() => {
        appResource.version().then(setInfo).catch(() => {});
    }, []);

    const isDev = info && info.version === "dev";

    const label = isDev ? t("app_update.dev_label", "build: dev") : (info && info.version);
    let tooltip = "";
    if (info) {
        const vars = {installed: info.version, latest: info.latest};
        if (isDev) tooltip = t("app_update.tip_dev", "Development build, update checks are off. Click for details.");
        else if (info.update_available) tooltip = t("app_update.tip_update", {defaultValue: "Update available: {{latest}} (installed {{installed}}). Click for notes and update steps.", ...vars});
        else if (info.latest) tooltip = t("app_update.tip_ok", {defaultValue: "Installed {{installed}}, latest {{latest}}. Click for details.", ...vars});
        else tooltip = t("app_update.tip_installed", {defaultValue: "Installed {{installed}}. Click for details.", ...vars});
    }

    const content = info && (
        <div className="text-sm">
            <div className="mb-1">{t("app_update.installed", "Installed")}: <b>{info.version}</b></div>
            {isDev && <div className="mb-2">{t("app_update.dev_build", "Development build, update checks are off.")}</div>}
            {!isDev && info.latest && <div className="mb-1">{t("app_update.latest", "Latest release")}: <b>{info.latest}</b></div>}
            {!isDev && info.latest && !info.update_available &&
                <div className="mb-2">{t("app_update.up_to_date", "You are running the latest version.")}</div>}
            {info.update_available && <>
                {info.notes &&
                    <div className="mt-3 mb-3">
                        <div className="font-bold">{t("app_update.notes", "What's new")}</div>
                        <div className="max-h-64 overflow-y-auto">{renderNotes(info.notes)}</div>
                    </div>}
                <div className="font-bold mb-1">{t("app_update.how_title", "How to update")}</div>
                {info.mode === "docker" && <>
                    <div className="mb-1">{t("app_update.how_docker", "Pull the new image, then start FSM again with fsm-start.sh from your fsm-data folder:")}</div>
                    <Command>{DOCKER_CMD}</Command>
                </>}
                {info.mode === "linux" && <>
                    <div className="mb-1">{t("app_update.how_linux", "Stop FSM, then run this in the install directory:")}</div>
                    <Command>{LINUX_CMD}</Command>
                </>}
                {info.mode === "windows" &&
                    <div className="mb-2">{t("app_update.how_windows", "Download the Windows archive from the release page and extract it over the current folder. Your data is kept.")}</div>}
                {info.release_url &&
                    <a className="underline" href={info.release_url} target="_blank" rel="noopener noreferrer">
                        {t("app_update.release_page", "Open the release page")}
                    </a>}
            </>}
        </div>
    );

    return (
        <div className="flex items-center gap-2 flex-wrap">
            <span className="text-dirty-white text-xl" title={t("main_title")}>FSM</span>
            {info &&
                <button className="text-sm text-dirty-white hover:text-orange cursor-pointer" title={tooltip} onClick={() => setIsOpen(true)}>
                    {label}
                </button>}
            {info && info.update_available &&
                <button className="text-xs px-2 py-0.5 bg-orange text-black font-bold cursor-pointer" title={tooltip} onClick={() => setIsOpen(true)}>
                    {t("app_update.available", "Update available")}
                </button>}
            <Modal
                title={t("app_update.title", "FSM version")}
                isOpen={isOpen}
                content={content}
                actions={<Button size="sm" onClick={() => setIsOpen(false)}>{t("app_update.close", "Close")}</Button>}
            />
        </div>
    );
};

export default VersionBadge;
