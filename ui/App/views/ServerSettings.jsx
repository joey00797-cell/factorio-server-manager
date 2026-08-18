import Panel from "../components/Panel";
import React, {useEffect, useState} from "react";
import settingsResource from "../../api/resources/settings";
import Input from "../components/Input";
import Label from "../components/Label";
import Checkbox from "../components/Checkbox";
import InputPassword from "../components/InputPassword";
import Button from "../components/Button";
import {useForm} from "react-hook-form";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import ServerScopeHeader from "../components/ServerScopeHeader";
import serverResource from "../../api/resources/server";

const ServerSettings = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();

    const [settings, setSettings] = useState();
    const [server, setServer] = useState(null);
    const [networkForm, setNetworkForm] = useState({name: "", bind_ip: "0.0.0.0", port: 34197, autostart: false, watchdog_interval: 0});
    const [isSavingNetwork, setIsSavingNetwork] = useState(false);
    const [numberInputs, setNumberInputs] = useState([]);

    const {register, handleSubmit, formState: {errors, isDirty}, control, reset} = useForm();

    const fetchSettings = async () => {
        const res = await settingsResource.server.list(serverId);
        setSettings(res);
        reset(res);
    };

    const fetchServer = async () => {
        const res = await serverResource.status(serverId);
        setServer(res);
        setNetworkForm({
            name: res.name || "",
            bind_ip: res.bindip || res.bind_ip || "0.0.0.0",
            port: res.port || 34197,
            autostart: !!res.autostart,
            watchdog_interval: res.watchdog_interval || 0
        });
    };

    const saveServerSettings = data => {
        data.tags = data.tags ? (Array.isArray(data.tags) ? data.tags : data.tags.split(',')) : [];
        data.admins = data.admins ? (Array.isArray(data.admins) ? data.admins : data.admins.split(',')) : [];

        numberInputs.forEach(numberInput => {
            data[numberInput] = parseInt(data[numberInput]);
        });

        Object.keys(settings).map(key => {
            if (key.startsWith("_comment")) {
                data[key] = settings[key];
            }
        });
       settingsResource.server.update(data, serverId)
           .then(() => {
               fetchSettings()
                   .then(() => window.flash(t("saved"), "green"))
           });
    }

    const saveNetworkSettings = async event => {
        event.preventDefault();
        const port = parseInt(networkForm.port, 10);
        if (!port || port < 1 || port > 65535) {
            window.flash(t("controls.port_error_message"), "red");
            return;
        }

        setIsSavingNetwork(true);
        try {
            await serverResource.update(serverId, {
                name: networkForm.name,
                bind_ip: networkForm.bind_ip || "0.0.0.0",
                port,
                autostart: !!networkForm.autostart,
                watchdog_interval: parseInt(networkForm.watchdog_interval) || 0
            });
            await fetchServer();
            window.flash(t("saved"), "green");
        } catch (err) {
            window.flash(err?.response?.data || err.message || t("server_settings.network_save_error", "Error saving server network settings."), "red");
        } finally {
            setIsSavingNetwork(false);
        }
    };

    useEffect(() => {
        fetchSettings();
        fetchServer();
    }, [serverId]);

    const formTypeField = (name, value, label = null) => {
        if (name.startsWith("_comment_")) {
            return null;
        }

        switch (typeof value) {
            case "undefined":
                break;
            case "function":
                break;
            case "symbol":
                break;
            case "bigint":
                break;
            case "number":

                if(numberInputs.indexOf(name) === -1) {
                    setNumberInputs(old => [...old, name])
                }

                return (
                    <>
                        <Label htmlFor={name} text={label}/>
                        <Input type="number" name={name} register={register} valueAsNumber="double" defaultValue={value} />
                    </>
                )
            case "string":
                if (name.includes("password")) {
                    return (
                        <>
                            <Label htmlFor={name} text={label}/>
                            <InputPassword name={name} register={register} defaultValue={value}/>
                        </>
                    )
                } else {
                    return (
                        <>
                            <Label htmlFor={name} text={label}/>
                            <Input name={name} register={register} defaultValue={value}/>
                        </>
                    )
                }
            case "boolean":
                return (
                    <Checkbox checked={value} text={label} register={register} name={name}/>
                )
            case "object":
                if (Array.isArray(value)) {
                    return (
                        <>
                            <Label htmlFor={name} text={label}/>
                            <Input name={name} register={register} defaultValue={value}/>
                        </>
                    )
                } else if (name.includes("visibility")) {
                    return (
                        <>
                            <Label text="Visibility"/>
                            <div className="flex">
                                {Object.keys(value).map(key => <div className="mr-4" key={`visibility-${key}`}>
                                    <Checkbox checked={value[key]} register={register} text={key} name={`visibility[${key}]`}/>
                                </div>)}
                            </div>
                        </>
                    )
                }
                break;
            default:
                return (
                    <>
                        <Label htmlFor={name} text={label}/>
                        <Input name={name} register={register} defaultValue={value}/>
                    </>
                )
        }
    }

    return (
        <>
            <ServerScopeHeader/>
            <form className="mb-4" onSubmit={saveNetworkSettings}>
                <Panel
                    title={t("server_settings.network_title", "Server network")}
                    content={
                        <div className="grid md:grid-cols-2 gap-4">
                            <div>
                                <Label htmlFor="server-name" text={t("name")}/>
                                <Input
                                    name="server-name"
                                    value={networkForm.name}
                                    onChange={event => setNetworkForm({...networkForm, name: event.target.value})}
                                />
                            </div>
                            <div>
                                <Label htmlFor="server-bind-ip" text={t("server_settings.bind_ip", "Bind IP")}/>
                                <Input
                                    name="server-bind-ip"
                                    value={networkForm.bind_ip}
                                    onChange={event => setNetworkForm({...networkForm, bind_ip: event.target.value})}
                                />
                            </div>
                            <div>
                                <Label htmlFor="server-port" text={t("controls.port")}/>
                                <Input
                                    name="server-port"
                                    type="number"
                                    min={1}
                                    max={65535}
                                    value={networkForm.port}
                                    onChange={event => setNetworkForm({...networkForm, port: event.target.value})}
                                />
                                {server?.running ? (
                                    <p className="text-sm italic mt-2 text-orange">
                                        {t("server_settings.port_restart_note", "Changing the port while running will require a restart.")}
                                    </p>
                                ) : null}
                            </div>
                            <div className="flex items-end">
                                <label className="block text-gray-500 font-bold">
                                    <input
                                        className="mr-2 leading-tight"
                                        type="checkbox"
                                        checked={networkForm.autostart}
                                        onChange={event => setNetworkForm({...networkForm, autostart: event.target.checked})}
                                    />
                                    <span className="text-sm">{t("controls.autostart")}</span>
                                </label>
                            </div>
                            <div className="flex flex-col gap-1">
                                <label className="block text-gray-500 font-bold text-sm">
                                    {t("servers.watchdog_interval", "Watchdog interval (sec)")}
                                </label>
                                <input
                                    className="shadow appearance-none border w-24 py-2 px-3 text-black"
                                    type="number"
                                    min="0"
                                    value={networkForm.watchdog_interval}
                                    onChange={e => setNetworkForm({...networkForm, watchdog_interval: e.target.value})}
                                />
                                <span className="text-xs text-gray-500">{t("servers.watchdog_hint", "0 = disabled. Reconnects RCON if server is running but unreachable.")}</span>
                            </div>
                            {server?.pending_restart ? (
                                <div className="md:col-span-2 text-orange font-bold">
                                    {t("servers.pending_restart", "restart pending")}
                                </div>
                            ) : null}
                        </div>
                    }
                    actions={
                        <Button isSubmit={true} type="success" isLoading={isSavingNetwork}>
                            {t("save")}
                        </Button>
                    }
                />
            </form>
            <form className="mb-4" onSubmit={handleSubmit(saveServerSettings)}>
                <Panel
                    title={t("server_settings.title")}
                    content={
                        <>
                            {settings && Object.keys(settings).map(key => {
                                if (key.startsWith("_comment_")) {
                                    return null;
                                }

                                const value = settings[key]
                                const label = key.replaceAll('_', ' ')
                                const comment = settings["_comment_" + key]

                                return (
                                    <div className="mb-4" key={`wrapper-${key}`}>
                                        {formTypeField(key, value, label)}
                                        <p className="text-sm italic">{comment}</p>
                                    </div>
                                )
                            })}
                        </>
                    }
                    actions={
                        <Button isSubmit={true} type="success" disabled={!isDirty}>{t("save")}</Button>
                    }
                />
            </form>
        </>
    )
}

export default ServerSettings;
