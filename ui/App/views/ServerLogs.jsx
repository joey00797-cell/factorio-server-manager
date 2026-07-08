import React, {useCallback, useEffect, useRef, useState} from "react";
import Panel from "../components/Panel";
import TabControl from "../components/Tabs/TabControl";
import Tab from "../components/Tabs/Tab";
import socket from "../../api/socket";
import log from "../../api/resources/log";
import serverResource from "../../api/resources/server";
import {useTranslation} from "react-i18next";
import {useParams} from "react-router-dom";
import ServerScopeHeader from "../components/ServerScopeHeader";

const POLL_INTERVAL_MS = 2000;

const useAutoScroll = () => {
    const [autoScroll, setAutoScroll] = useState(true);
    const autoScrollRef = useRef(true);
    const bottomRef = useRef(null);
    const toggle = (val) => { setAutoScroll(val); autoScrollRef.current = val; };
    const scrollToBottom = useCallback(() => {
        if (autoScrollRef.current && bottomRef.current) bottomRef.current.scrollIntoView({behavior: "smooth"});
    }, []);
    return {autoScroll, toggle, bottomRef, scrollToBottom};
};

const AutoScrollCheckbox = ({autoScroll, toggle, id}) => {
    const {t} = useTranslation();
    return (
        <div className="flex items-center gap-2 mb-2">
            <input type="checkbox" id={id} checked={autoScroll} onChange={e => toggle(e.target.checked)}/>
            <label htmlFor={id} className="text-sm cursor-pointer">{t("console.autoscroll", "Auto-scroll")}</label>
        </div>
    );
};

const ConsoleTab = ({serverId}) => {
    const {t} = useTranslation();
    const [logs, setLogs] = useState([]);
    const [serverStatus, setServerStatus] = useState({running: false});
    const consoleInput = useRef(null);
    const {autoScroll, toggle, bottomRef, scrollToBottom} = useAutoScroll();

    useEffect(() => {
        serverResource.status(serverId).then(setServerStatus);
        log.tail(serverId).then(lines => { if (Array.isArray(lines)) setLogs(lines); }).catch(() => {});
        const appendLog = line => setLogs(lines => [...lines, line]);
        const room = serverId ? `servers:${serverId}:gamelog` : "gamelog";
        socket.on(room, appendLog);
        socket.emit("log subscribe", serverId);
        consoleInput.current?.focus();
        return () => { socket.off(room, appendLog); socket.emit("log unsubscribe", serverId); };
    }, [serverId]);

    useEffect(() => { scrollToBottom(); }, [logs, scrollToBottom]);

    if (!serverStatus.running) return <p className="text-red-light pt-4">{t("console.error")}</p>;

    return (
        <>
            <AutoScrollCheckbox autoScroll={autoScroll} toggle={toggle} id="autoscroll-console"/>
            <ul className="max-h-[60vh] overflow-y-auto">
                {logs?.map((line, i) => (<li key={i}>{line}</li>))}
                <li ref={bottomRef}/>
            </ul>
            <input type="text"
                className="shadow appearance-none border w-full py-2 px-3 text-black mt-2"
                ref={consoleInput}
                onKeyDown={e => {
                    if (e.key === "Enter" && socket) {
                        socket.emit("command send", consoleInput.current.value, serverId);
                        consoleInput.current.value = "";
                    }
                }}
            />
        </>
    );
};

const ServerLogsTab = ({serverId}) => {
    const [logs, setLogs] = useState([]);
    const {autoScroll, toggle, bottomRef, scrollToBottom} = useAutoScroll();

    useEffect(() => {
        let cancelled = false;
        let inFlight = false;
        const poll = async () => {
            if (inFlight) return;
            inFlight = true;
            try {
                const lines = await log.tail(serverId);
                if (!cancelled) setLogs(Array.isArray(lines) ? lines : []);
            } catch (err) {
                console.error("Error refreshing server logs", err);
            } finally { inFlight = false; }
        };
        poll();
        const interval = setInterval(poll, POLL_INTERVAL_MS);
        return () => { cancelled = true; clearInterval(interval); };
    }, [serverId]);

    useEffect(() => { scrollToBottom(); }, [logs, scrollToBottom]);

    return (
        <>
            <AutoScrollCheckbox autoScroll={autoScroll} toggle={toggle} id="autoscroll-serverlogs"/>
            <div className="max-h-[60vh] overflow-y-auto">
                <pre className="whitespace-pre-wrap font-mono text-sm">{logs.join("\n")}</pre>
                <div ref={bottomRef}/>
            </div>
        </>
    );
};

const FsmLogsTab = () => {
    const [logs, setLogs] = useState([]);
    const {autoScroll, toggle, bottomRef, scrollToBottom} = useAutoScroll();

    useEffect(() => {
        let cancelled = false;
        let inFlight = false;
        const poll = async () => {
            if (inFlight) return;
            inFlight = true;
            try {
                const lines = await log.fsmTail();
                if (!cancelled) setLogs(lines || []);
            } catch (err) {
                console.error("Error refreshing FSM logs", err);
            } finally { inFlight = false; }
        };
        poll();
        const interval = setInterval(poll, POLL_INTERVAL_MS);
        return () => { cancelled = true; clearInterval(interval); };
    }, []);

    useEffect(() => { scrollToBottom(); }, [logs, scrollToBottom]);

    return (
        <>
            <AutoScrollCheckbox autoScroll={autoScroll} toggle={toggle} id="autoscroll-fsmlogs"/>
            <div className="max-h-[60vh] overflow-y-auto">
                <pre className="whitespace-pre-wrap font-mono text-sm">{logs.join("\n")}</pre>
                <div ref={bottomRef}/>
            </div>
        </>
    );
};

const ServerLogs = () => {
    const {t} = useTranslation();
    const {serverId} = useParams();
    return (
        <>
            <ServerScopeHeader/>
            <Panel
                title={t("logs.title")}
                content={
                    <TabControl>
                        <Tab title={t("console.title")}><ConsoleTab serverId={serverId}/></Tab>
                        <Tab title={t("logs.title")}><ServerLogsTab serverId={serverId}/></Tab>
                        <Tab title={t("fsm_logs.title", "FSM Logs")}><FsmLogsTab/></Tab>
                    </TabControl>
                }
            />
        </>
    );
};

export default ServerLogs;
