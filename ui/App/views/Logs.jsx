import React, {useCallback, useEffect, useRef, useState} from "react";
import log from "../../api/resources/log";
import Panel from "../components/Panel";
import { useTranslation } from "react-i18next";
import {useParams} from "react-router-dom";
import ServerScopeHeader from "../components/ServerScopeHeader";

const POLL_INTERVAL_MS = 2000;

const Logs = () => {
    const { t } = useTranslation();
    const {serverId} = useParams();

    const [logs, setLogs] = useState([]);
    const [autoScroll, setAutoScroll] = useState(true);
    const autoScrollRef = useRef(true);
    const logContainerRef = useRef(null);
    const bottomRef = useRef(null);

    const scrollToBottom = useCallback(() => {
        if (!autoScrollRef.current) return;
        if (logContainerRef.current) {
            logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
        }
        if (bottomRef.current) {
            bottomRef.current.scrollIntoView({block: "end"});
        }
    }, []);

    useEffect(() => {
        let cancelled = false;
        let inFlight = false;

        const poll = async () => {
            if (inFlight) {
                return;
            }
            inFlight = true;
            try {
                const lines = await log.tail(serverId);
                if (!cancelled) {
                    setLogs(Array.isArray(lines) ? lines : []);
                }
            } catch (err) {
                console.error("Error refreshing server logs", err);
            } finally {
                inFlight = false;
            }
        };

        poll();
        const interval = setInterval(poll, POLL_INTERVAL_MS);
        return () => {
            cancelled = true;
            clearInterval(interval);
        };
    }, [serverId]);

    useEffect(() => {
        scrollToBottom();
    }, [logs, scrollToBottom]);

    return (
        <>
            <ServerScopeHeader/>
            <Panel
                title={t("logs.title")}
                content={
                    <>
                    <div className="flex items-center gap-2 mb-2">
                        <input type="checkbox" id="autoscroll-logs" checked={autoScroll} onChange={e => { setAutoScroll(e.target.checked); autoScrollRef.current = e.target.checked; }}/>
                        <label htmlFor="autoscroll-logs" className="text-sm cursor-pointer">Auto-scroll</label>
                    </div>
                    <div
                        ref={logContainerRef}
                        className="max-h-[70vh] overflow-y-auto"
                    >
                        <pre className="whitespace-pre-wrap font-mono text-sm">
                            {logs.join("\n")}
                        </pre>
                        <div ref={bottomRef}/>
                    </div>
                    </>
                }
            />
        </>
    );
}

export default Logs;
