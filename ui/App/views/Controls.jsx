import React, {useEffect, useState} from "react";
import Panel from "../components/Panel";
import Button from "../components/Button";
import server from "../../api/resources/server";
import savesResource from "../../api/resources/saves";
import {useForm} from "react-hook-form";
import Select from "../components/Select";
import Input from "../components/Input";
import Error from "../components/Error";
import { useTranslation } from "react-i18next";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import socket from "../../api/socket";
import {faToggleOn, faToggleOff} from "@fortawesome/free-solid-svg-icons";

const Controls = ({serverStatus}) => {
    const { t } = useTranslation();
    const factorioVersion = serverStatus.fac_version ? serverStatus.fac_version : t("controls.unknown");
    const isFactorioInstalled = serverStatus.fac_version && serverStatus.fac_version !== "0.0.0.0";
    const [availableVersions, setAvailableVersions] = useState({});
    const [isInstalling, setIsInstalling] = useState(false);
    const [installProgress, setInstallProgress] = useState(0);
    const [installCurrent, setInstallCurrent] = useState(0);
    const [installTotal, setInstallTotal] = useState(0);
    const [isExtracting, setIsExtracting] = useState(false);
    const [selectedVersion, setSelectedVersion] = useState('stable');
    const [autostart, setAutostart] = useState(false);
    const [saves, setSaves] = useState([]);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isStopping, setIsStopping] = useState(false);
    const [isStarting, setIsStarting] = useState(false);
    const [isKilling, setIsKilling] = useState(false);

    const { handleSubmit, reset, register, formState: {errors} } = useForm();

    useEffect(() => {
        server.availableVersions()
            .then(res => setAvailableVersions(res));
        savesResource.list(true)
            .then(res => {
                setSaves(res);
                if (res.length > 0) setIsDisabled(undefined);
                reset();
            });
        // Читаем состояние автостарта
        fetch("/api/autostart")
            .then(r => r.json())
            .then(data => setAutostart(data.autostart))
            .catch(() => {});
    }, []);

    const startServer = async (data) => {
        setIsStarting(true);
        await server.start(data.ip, parseInt(data.port), data.save);
    }

    const stopServer = async () => {
        setIsStopping(true);
        await server.stop();
    }

    const killServer = async () => {
        setIsKilling(true);
        await server.kill();
    }

    useEffect(() => {
        socket.emit('server version subscribe');
        server.installStatus().then(status => {
            if (status && status.installing) {
                setIsInstalling(true);
                setInstallProgress(status.progress || 0);
            }
        }).catch(() => {});
        const handleVersionMsg = (msg) => {
            try {
                const data = JSON.parse(msg);
                if (data.type === 'download_progress') {
                    setInstallProgress(data.percent);
                    setInstallCurrent(Math.round(data.current / 1024 / 1024 * 10) / 10);
                    setInstallTotal(Math.round(data.total / 1024 / 1024 * 10) / 10);
                } else if (data.type === 'extracting') {
                    setInstallProgress(100);
                    setIsExtracting(true);
                } else if (data.type === 'install_complete') {
                    setIsInstalling(false);
                    setIsExtracting(false);
                    setInstallProgress(0);
                    setInstallCurrent(0);
                    setInstallTotal(0);
                    window.flash('Factorio ' + data.version + ' installed!', 'green');
                } else if (data.type === 'install_error') {
                    setIsInstalling(false);
                    setIsExtracting(false);
                    setInstallProgress(0);
                    window.flash('Install failed: ' + data.error, 'red');
                }
            } catch(e) {}
        };
        socket.on('server_version', handleVersionMsg);
        return () => socket.off('server_version', handleVersionMsg);
    }, []);

    const installVersion = async () => {
        setIsInstalling(true);
        setInstallProgress(0);
        try {
            await server.installVersion(selectedVersion);
        } catch(e) {
            if (e?.response?.status !== 409) {
                setIsInstalling(false);
                window.flash('Install failed', 'red');
            }
        }
    }

    const versionLabel = (type) => {
        const v = availableVersions?.[type]?.headless;
        return v ? `${type} (${v})` : type;
    }

    return (
        <form onSubmit={handleSubmit(startServer)}>
        {!isFactorioInstalled && (
            <div className="mb-4 p-3 bg-red bg-opacity-20 border border-red rounded text-red-light font-bold">
                ⚠ {t("controls.factorio_not_installed")}
            </div>
        )}
        <Panel
            title={t("controls.title")}
            content={
                <div className="lg:flex">
                    {/* Статус + Автостарт */}
                    <div className="lg:flex-1 mb-2 min-w-0">
                        <div className="font-bold">{t("controls.status")}</div>
                        <div>{serverStatus.running ? t("controls.running") : t("controls.stopped")}</div>
                        <div className="mt-2 flex items-center gap-2">
                            <FontAwesomeIcon
                                className={`cursor-pointer text-xl ${autostart ? 'text-green' : 'text-red'}`}
                                icon={autostart ? faToggleOn : faToggleOff}
                                onClick={() => {
                                    const newVal = !autostart;
                                    setAutostart(newVal);
                                    fetch("/api/autostart", {
                                        method: "POST",
                                        headers: {"Content-Type": "application/json"},
                                        body: JSON.stringify({autostart: newVal})
                                    });
                                }}
                            />
                            <span className="text-sm">{t("controls.autostart")}</span>
                        </div>
                    </div>

                    {serverStatus.running ? <>
                        <div className="lg:flex-1 mb-2 min-w-0">
                            <div className="font-bold">IP</div>
                            <div>{serverStatus.bindip}</div>
                        </div>
                        <div className="lg:flex-1 mb-2 min-w-0">
                            <div className="font-bold">{t("controls.port")}</div>
                            <div>{serverStatus.port}</div>
                        </div>
                        <div className="lg:flex-1 mb-2 min-w-0">
                            <div className="font-bold">{t("controls.f_version")}</div>
                            <div>{factorioVersion}</div>
                        </div>
                        <div className="lg:flex-1 mb-2 min-w-0">
                            <div className="font-bold">{t("controls.save")}</div>
                            <div>{serverStatus.savefile}</div>
                        </div>
                    </> : <>
                        <div className="lg:flex-1 mb-2 mr-0 lg:mr-4 min-w-0">
                            <div className="font-bold">IP</div>
                            <Input
                                defaultValue={"0.0.0.0"}
                                disabled={isDisabled}
                                register={register('ip',{required: true, pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/})}
                            />
                            <Error error={errors.ip} message={t("controls.IP_error_message")}/>
                        </div>
                        <div className="lg:flex-1 mb-2 mr-0 lg:mr-4 min-w-0">
                            <div className="font-bold">{t("controls.port")}</div>
                            <Input
                                type="number"
                                min={1}
                                max={65535}
                                defaultValue={"34197"}
                                disabled={isDisabled}
                                register={register('port',{required: true, min: 1, max: 65535})}
                            />
                            <Error error={errors.port} message={t("controls.port_error_message")}/>
                        </div>
                        <div className="lg:flex-1 mb-2 mr-0 lg:mr-4 min-w-0">
                            <div className="font-bold">{t("controls.f_version")}</div>
                            <select
                                className="w-full border rounded px-2 py-2 text-black"
                                value={selectedVersion}
                                onChange={e => setSelectedVersion(e.target.value)}
                                disabled={serverStatus.running || isInstalling}
                            >
                                <option value="stable">{versionLabel('stable')}</option>
                                <option value="experimental">{versionLabel('experimental')}</option>
                            </select>
                        </div>
                        <div className="lg:flex-1 mb-2 min-w-0">
                            <div className="font-bold">{t("controls.save")}</div>
                            <div className="relative">
                                <Select
                                    register={register('save',{required: true})}
                                    defaultValue={saves.find((save) => save.name.startsWith('Load Latest'))?.name}
                                    disabled={isDisabled}
                                    options={saves.map(save => ({value: save.name, name: save.name}))}
                                />
                                <Error error={errors.save} message={t("controls.save_error_message")}/>
                            </div>
                        </div>
                    </>}
                </div>
            }
            actions={
                <>
                <div className="md:flex items-center gap-2">
                    {serverStatus.running ? <>
                        <Button onClick={stopServer} isLoading={isStopping} isDisabled={isKilling} size="sm" className="w-full md:w-auto mb-2 md:mb-0" type="default">{t("controls.save&stop")}</Button>
                        <Button onClick={killServer} isLoading={isKilling} isDisabled={isStopping} size="sm" type="danger" className="w-full md:w-auto">{t("controls.kill_server")}</Button>
                    </> : <>
                        <Button isSubmit={true} isDisabled={isDisabled || isInstalling || !isFactorioInstalled} isLoading={isStarting} size="sm" type="success" className="w-full md:w-auto">{t("controls.start_server")}</Button>
                        <Button onClick={installVersion} isLoading={isInstalling} isDisabled={serverStatus.running} size="sm" type="default" className="w-full md:w-auto">{t("controls.install_factorio")}</Button>
                    </>
                    }
                </div>
                {isInstalling && (
                    <div className="mt-4 mx-2">
                        <div className="w-full bg-gray-dark rounded h-3">
                            <div className="bg-orange h-3 rounded transition-all duration-300" style={{width: `${installProgress}%`}}/>
                        </div>
                        <p className="text-sm text-gray-light mt-1">{isExtracting ? t("controls.extracting") : t("controls.downloading", {current: installCurrent, total: installTotal})}</p>
                    </div>
                )}
                </>
            }
        />
        </form>
    )
};

export default Controls;
