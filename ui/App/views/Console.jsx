import Panel from "../components/Panel";
import React, {useCallback, useEffect, useRef, useState} from "react";
import socket from "../../api/socket";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import serverResource from "../../api/resources/server";
import log from "../../api/resources/log";
import ServerScopeHeader from "../components/ServerScopeHeader";

const Console = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();

    const [logs, setLogs] = useState([]);
    const [serverStatus, setServerStatus] = useState({running: false});
    const [autoScroll, setAutoScroll] = useState(true);
    const autoScrollRef = useRef(true);
    const consoleInput = useRef(null);
    const bottomRef = useRef(null);

    useEffect(() => {
        serverResource.status(serverId).then(setServerStatus);
        log.tail(serverId).then(lines => {
            if (Array.isArray(lines)) setLogs(lines);
        }).catch(() => {});

        const appendLog = line => {
            setLogs(lines => [...lines, line]);
        }

        const room = serverId ? `servers:${serverId}:gamelog` : 'gamelog';
        socket.on(room, appendLog)
        socket.emit('log subscribe', serverId)
        consoleInput.current?.focus();

        return () => {
            socket.off(room, appendLog);
            socket.emit("log unsubscribe", serverId)
        }
    }, [serverId]);

    useEffect(() => {
        if (autoScrollRef.current && bottomRef.current) bottomRef.current.scrollIntoView({behavior: "smooth"});
    }, [logs]);

    return (
        <>
            <ServerScopeHeader/>
            <Panel
                title={t("console.title")}
                content={
                    serverStatus.running
                        ? <>
                            <div className="flex items-center gap-2 mb-2">
                                <input type="checkbox" id="autoscroll-console" checked={autoScroll} onChange={e => { setAutoScroll(e.target.checked); autoScrollRef.current = e.target.checked; }}/>
                                <label htmlFor="autoscroll-console" className="text-sm cursor-pointer">Auto-scroll</label>
                            </div>
                            <ul>
                                {logs?.map((log, i) => (<li key={i}>{log}</li>))}
                                <li ref={bottomRef}/>
                            </ul>
                            <input type="text"
                                   className="shadow appearance-none border w-full py-2 px-3 text-black"
                                   ref={consoleInput}
                                   onKeyDown={e => {
                                       if (e.key === "Enter" && socket) {
                                           socket.emit("command send", consoleInput.current.value, serverId);
                                           consoleInput.current.value = ""
                                       }
                                   }}
                            />
                        </>
                        : <p className="text-red-light pt-4">
                            {t("console.error")}
                        </p>
                }
            />
        </>
    )
}

export default Console;
