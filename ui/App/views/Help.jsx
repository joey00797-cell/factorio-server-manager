import Panel from "../components/Panel";
import React, {useEffect, useState} from "react";
import { useTranslation, Trans } from "react-i18next";
import client from "../../api/client";
import serverResource from "../../api/resources/server";

const REPO = "https://github.com/joey00797-cell/factorio-server-manager";

const HOWTO = [
    ["Create a server", "Servers tab -> create form. Pick a version and a name. Bind IP is always saved as 0.0.0.0, the real address is shown for information only. Each server gets its own instance folder with saves, mods, config and logs."],
    ["Upload a save", "Open the server, Saves tab, upload a .zip. Select it, then use Sync on the Mods tab to pull the mods the save needs."],
    ["Sync mods from a save", "Sync button has two modes. Match save: enables the mods from the save, disables all others and applies automatically. Add from save: only enables what the save needs, nothing is disabled. If the library has another version of a mod, the one from the save wins in the manifest."],
    ["Mod library and the trash can", "The library keeps every uploaded or downloaded version. Toggles and Apply only change the server manifest and the instance folder. Only the trash can removes a mod from the library physically."],
    ["Apply preview", "Any manifest change shows an Apply window: To Add, To Remove, To Enable, To Disable. Untick items to skip them. Cancel resets the manifest to what is deployed. The server must be stopped to apply."],
    ["Backups", "Backups are stored outside instances (fsm-data/backups/<serverID>), so they survive a wipe unless you tick the backups option. Backups of other servers are shown dimmed and can be restored into the current one. Schedule and retention are configured on the same tab."],
    ["Watchdog", "Set an interval in server settings (seconds, 0 = off). FSM checks the server every few seconds and restarts it if it died."],
];

const PARTS = [
    ["saves", "Saves"],
    ["mods", "Mods (instance folder + manifest; library stays)"],
    ["config", "Config (config.ini, settings, admin list)"],
    ["logs", "Logs"],
];

const Help = () => {
    const { t } = useTranslation();
    const [servers, setServers] = useState([]);
    const [serverId, setServerId] = useState("");
    const [checks, setChecks] = useState({saves: false, mods: false, config: false, logs: false, backups: false, full: false});
    const [sure, setSure] = useState(false);
    const [confirm, setConfirm] = useState("");
    const [busy, setBusy] = useState(false);

    useEffect(() => {
        serverResource.list().then(data => {
            const list = Array.isArray(data) ? data : (data && data.servers) || [];
            setServers(list);
            if (list.length > 0) setServerId(String(list[0].id));
        }).catch(() => {});
    }, []);

    const target = servers.find(s => String(s.id) === serverId);
    const anyChecked = Object.values(checks).some(Boolean);
    const canWipe = !!target && anyChecked && confirm === target.name && (!checks.backups || sure) && !busy;

    const wipe = async () => {
        setBusy(true);
        try {
            const res = await client.post(`/api/servers/${serverId}/wipe`, checks);
            const d = res.data || {};
            window.flash("Wiped: " + Object.keys(d).map(k => `${k} (${d[k]})`).join(", "), "green");
            setConfirm("");
            setSure(false);
            if (checks.full) {
                const data = await serverResource.list();
                const list = Array.isArray(data) ? data : (data && data.servers) || [];
                setServers(list);
                setServerId(list.length > 0 ? String(list[0].id) : "");
                setChecks({saves: false, mods: false, config: false, logs: false, backups: false, full: false});
            }
        } catch (e) {
            const msg = e.response && e.response.data ? JSON.stringify(e.response.data) : "";
            window.flash("Wipe failed " + msg, "red");
        } finally {
            setBusy(false);
        }
    };

    return (
        <Panel
            title={t("help.title")}
            content={
                <>
                    <h1 className="text-xl text-dirty-white">{t("help.fsm")}</h1>
                    <p className="mb-2">{t("help.fsm_content")}</p>

                    <h2 className="text-dirty-white">{t("help.bugs_help")}</h2>
                    <p className="mb-4">Please use the <a className="text-blue hover:text-blue-light" target="_blank" href={REPO + "/issues"}>GitHub repository</a> to report bugs or seek for help.</p>

                    <h1 className="mb-1 text-xl text-dirty-white">How-to</h1>
                    <div className="mb-4">
                        {HOWTO.map(([title, text]) => (
                            <details key={title} className="mb-1 bg-gray-dark px-2 py-1">
                                <summary className="cursor-pointer select-none hover:text-orange">{title}</summary>
                                <p className="mt-1 mb-1">{text}</p>
                            </details>
                        ))}
                    </div>

                    <h1 className="mb-1 text-xl text-dirty-white">{t("help.helpful_resources")}</h1>
                    <p className="mb-4"><a className="text-blue hover:text-blue-light" target="_blank" href="https://wiki.factorio.com/Multiplayer">{t("help.factorio_link_text")}</a></p>

                    <details className="border border-red px-2 py-1">
                        <summary className="cursor-pointer select-none text-red font-bold">Danger zone: wipe server instance</summary>
                        <div className="mt-2">
                            <p className="mb-2">Deletes the selected data of one server. The server must be stopped. Backups are kept unless you tick the backups option. The mod library is never touched.</p>
                            <div className="mb-2">
                                <select className="text-black" value={serverId} onChange={e => { setServerId(e.target.value); setConfirm(""); }}>
                                    {servers.map(s => <option key={s.id} value={s.id}>{s.name} (#{s.id})</option>)}
                                </select>
                            </div>
                            {PARTS.map(([key, label]) => (
                                <label key={key} className="block cursor-pointer">
                                    <input type="checkbox" className="mr-2" checked={checks[key] && !checks.full} disabled={checks.full}
                                           onChange={e => setChecks({...checks, [key]: e.target.checked})}/>
                                    {label}
                                </label>
                            ))}
                            <label className="block cursor-pointer mt-2 text-red font-bold">
                                <input type="checkbox" className="mr-2" checked={checks.full}
                                       onChange={e => setChecks({...checks, full: e.target.checked})}/>
                                Delete server completely (instance, manifest, catalog entry)
                            </label>
                            <label className="block cursor-pointer text-red">
                                <input type="checkbox" className="mr-2" checked={checks.backups}
                                       onChange={e => { setChecks({...checks, backups: e.target.checked}); setSure(false); }}/>
                                Also wipe backups of this server
                            </label>
                            {checks.backups && (
                                <label className="block cursor-pointer ml-6 mt-1 border border-red px-2 py-1 text-red font-bold">
                                    <input type="checkbox" className="mr-2" checked={sure} onChange={e => setSure(e.target.checked)}/>
                                    Are you sure? Backups will be deleted permanently and cannot be restored.
                                </label>
                            )}
                            <p className="mt-2 mb-1">Type the server name{target ? <b> {target.name} </b> : " "}to confirm:</p>
                            <input className="text-black px-1 mr-2" value={confirm} onChange={e => setConfirm(e.target.value)}/>
                            <button disabled={!canWipe} onClick={wipe}
                                    className={`px-2 py-1 font-bold text-black ${canWipe ? "bg-red cursor-pointer" : "bg-gray opacity-50 cursor-not-allowed"}`}>
                                {busy ? "Working..." : (checks.full ? "Delete server" : "Wipe")}
                            </button>
                        </div>
                    </details>
                </>
            }
        />
    )
}

export default Help;
