import React, {useEffect, useState} from "react";
import { useTranslation } from 'react-i18next';
import Panel from "../components/Panel";
import Button from "../components/Button";
import server from "../../api/resources/server";
import socket from "../../api/socket";

const ServerVersion = ({serverStatus}) => {

    const { t } = useTranslation('serverVersion');
    const [currentVersion, setCurrentVersion] = useState(null);
    const [availableVersions, setAvailableVersions] = useState([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isInstalling, setIsInstalling] = useState(false);
    const [installVersion, setInstallVersion] = useState(null);
    const [progress, setProgress] = useState(0);
    const [error, setError] = useState(null);
    const [showAll, setShowAll] = useState(false);
    const [allVersions, setAllVersions] = useState([]);

    const fetchData = async () => {
        setIsLoading(true);
        setError(null);
        try {
            const current = await server.version.current();
            setCurrentVersion(current.version);
        } catch (e) {
            setError(t('fetchFailed'));
        }
        try {
            const available = await server.version.available();
            setAvailableVersions(available || []);
        } catch (e) {
            setError(t('fetchFailed'));
        }
        setIsLoading(false);
    };

    useEffect(() => {
        fetchData();
        socket.emit('server version subscribe');

        (async () => {
            const status = await server.version.installStatus();
            if (status && status.installing) {
                setIsInstalling(true);
                setInstallVersion(status.version);
                setProgress(status.progress);
            }
        })();

        const handleVersionMessage = (msg) => {
            try {
                const data = JSON.parse(typeof msg === 'string' ? msg : JSON.stringify(msg));
                if (data.type === 'download_progress') {
                    setProgress(data.percent);
                } else if (data.type === 'install_complete') {
                    setIsInstalling(false);
                    setProgress(0);
                    setCurrentVersion(data.version);
                    setInstallVersion(null);
                    window.flash(t('installSuccess').replace('{version}', data.version), "green");
                } else if (data.type === 'install_error') {
                    setIsInstalling(false);
                    setProgress(0);
                    setInstallVersion(null);
                    setError(data.error || 'Unknown error');
                }
            } catch (e) {}
        };

        socket.on('server_version', handleVersionMessage);
        return () => socket.off('server_version', handleVersionMessage);
    }, []);

    const toggleAllVersions = async () => {
        if (showAll) {
            setShowAll(false);
            return;
        }
        if (allVersions.length === 0) {
            try {
                const list = await server.version.list();
                setAllVersions(list || []);
            } catch (e) {
                setError(t('fetchFailed'));
                return;
            }
        }
        setShowAll(true);
    };

    const displayVersions = showAll && allVersions.length > 0 ? allVersions : availableVersions;

    const stableReleases = displayVersions.filter(r => r.stable);
    const experimentalReleases = displayVersions.filter(r => !r.stable);

    const renderVersionRow = (release) => (
        <div key={release.version} className="flex items-center justify-between p-3">
            <div className="flex items-center space-x-2">
                <span className="text-dirty-white">{release.version}</span>
                {!showAll && release.stable ? (
                    <span className="text-xs bg-green text-black px-1 rounded">{t('stable')}</span>
                ) : (!showAll ? (
                    <span className="text-xs bg-orange text-black px-1 rounded">{t('experimental')}</span>
                ) : null)}
                {!showAll && displayVersions.indexOf(release) === 0 && (
                    <span className="text-xs bg-blue text-white px-1 rounded">{t('latest')}</span>
                )}
            </div>
            <Button
                size="sm"
                isLoading={isInstalling && installVersion === release.version}
                isDisabled={isInstalling || currentVersion === release.version}
                onClick={() => handleInstall(release.version)}
            >
                {currentVersion === release.version ? t('current') : t('install')}
            </Button>
        </div>
    );

    const handleInstall = async (version) => {
        if (serverStatus && serverStatus.running) {
            window.flash(t('serverMustBeStopped'), "red");
            return;
        }
        setInstallVersion(version);
        setIsInstalling(true);
        setProgress(0);
        setError(null);
        try {
            await server.version.install(version);
        } catch (e) {
            setError(t('installFailed').replace('{error}', e.response?.data?.error || e.message || ''));
            setIsInstalling(false);
            setInstallVersion(null);
        }
    };

    const hasUpdate = currentVersion && availableVersions.length > 0
        && availableVersions[0].version !== currentVersion;

    return (
        <Panel
            title={t('serverVersion')}
            content={
                <>
                    {isLoading ? (
                        <p className="text-gray-light">{t('checkingVersion')}</p>
                    ) : (
                        <>
                            <div className="mb-6">
                                <h2 className="text-dirty-white text-lg mb-2">{t('installedVersion')}</h2>
                                <div className="bg-black rounded p-4 border border-gray-light">
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <span className="text-dirty-white font-bold">{currentVersion || t('unknown', { ns: 'common' })}</span>
                                        </div>
                                        <div>
                                            {hasUpdate ? (
                                                <span className="text-orange text-sm">{t('updateAvailable')}</span>
                                            ) : (
                                                <span className="text-green text-sm">{t('upToDate')}</span>
                                            )}
                                        </div>
                                    </div>
                                    {isInstalling && (
                                        <div className="mt-3">
                                            <div className="w-full bg-gray-dark rounded h-2">
                                                <div
                                                    className="bg-orange h-2 rounded transition-all duration-300"
                                                    style={{width: `${progress}%`}}
                                                />
                                            </div>
                                            <p className="text-sm text-gray-light mt-1">
                                                {t('downloadProgress').replace('{percent}', progress)}
                                            </p>
                                        </div>
                                    )}
                                    {error && (
                                        <p className="text-red text-sm mt-2">{error}</p>
                                    )}
                                </div>
                            </div>

                            <div>
                                <h2 className="text-dirty-white text-lg mb-2">{t('availableVersions')}</h2>
                                <div className="bg-black rounded border border-gray-light">
                                    {displayVersions.length === 0 ? (
                                        <p className="p-4 text-gray-light">{t('noInternet')}</p>
                                    ) : showAll ? (
                                        <div>
                                            {stableReleases.length > 0 && (
                                                <div className="mb-4">
                                                    <div className="px-3 py-2 bg-gray-dark">
                                                        <span className="text-green font-bold">{t('stable')}</span>
                                                        <span className="text-gray-light text-sm ml-2">({stableReleases.length})</span>
                                                    </div>
                                                    <div className="divide-y divide-gray-light">
                                                        {stableReleases.map(r => renderVersionRow(r))}
                                                    </div>
                                                </div>
                                            )}
                                            {experimentalReleases.length > 0 && (
                                                <div>
                                                    <div className="px-3 py-2 bg-gray-dark">
                                                        <span className="text-orange font-bold">{t('experimental')}</span>
                                                        <span className="text-gray-light text-sm ml-2">({experimentalReleases.length})</span>
                                                    </div>
                                                    <div className="divide-y divide-gray-light">
                                                        {experimentalReleases.map(r => renderVersionRow(r))}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                    ) : (
                                        <div className="divide-y divide-gray-light">
                                            {displayVersions.map((release, i) => renderVersionRow(release))}
                                        </div>
                                    )}
                                </div>
                            </div>

                            <div className="mt-2 text-center">
                                <Button size="sm" type="default" onClick={toggleAllVersions}>
                                    {showAll ? t('showLatest') : t('showAll')}
                                </Button>
                            </div>

                            {hasUpdate && !isInstalling && (
                                <div className="mt-4 text-center">
                                    <Button
                                        type="success"
                                        onClick={() => handleInstall(availableVersions[0].version)}
                                    >
                                        {t('updateTo').replace('{version}', availableVersions[0].version)}
                                    </Button>
                                </div>
                            )}
                        </>
                    )}
                </>
            }
        />
    )
};

export default ServerVersion;
