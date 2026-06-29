import EventEmitter from "events";

const bus = new EventEmitter();

const ws_scheme = window.location.protocol === "https:" ? "wss" : "ws";

function connect() {
    const socket = new WebSocket(ws_scheme + "://" + window.location.host + "/ws");

    function logSubscribeEvent(serverId) {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "subscribe",
                        value: serverId ? `servers:${serverId}:gamelog` : "gamelog"
                    }
                }
            )
        );
    }

    function logUnsubscribeEvent(serverId) {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "unsubscribe",
                        value: serverId ? `servers:${serverId}:gamelog` : "gamelog"
                    }
                }
            )
        );
    }

    function serverVersionSubscribeEvent() {
        socket.send(JSON.stringify({room_name: "", controls: {type: "subscribe", value: "server_version"}}));
    }

    function serverStatusSubscribeEvent() {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "subscribe",
                        value: "server_status"
                    }
                }
            )
        );
    }

    function serversSubscribeEvent() {
        socket.send(JSON.stringify({room_name: "", controls: {type: "subscribe", value: "servers"}}));
    }

    function serversUnsubscribeEvent() {
        socket.send(JSON.stringify({room_name: "", controls: {type: "unsubscribe", value: "servers"}}));
    }

    function commandSendEvent(command, serverId) {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "command",
                        value: serverId ? JSON.stringify({serverId, command}) : command
                    }
                }
            )
        );
    }

    function modsSyncSubscribeEvent(serverId) {
        socket.send(JSON.stringify({room_name: "", controls: {type: "subscribe", value: serverId ? `servers:${serverId}:mods_sync` : "mods_sync"}}));
    }

    function modsSyncUnsubscribeEvent(serverId) {
        socket.send(JSON.stringify({room_name: "", controls: {type: "unsubscribe", value: serverId ? `servers:${serverId}:mods_sync` : "mods_sync"}}));
    }

    function registerEventEmitter() {
        bus.on('log subscribe', logSubscribeEvent);
        bus.on('log unsubscribe', logUnsubscribeEvent);
        bus.on('server status subscribe', serverStatusSubscribeEvent);
        bus.on('server version subscribe', serverVersionSubscribeEvent);
        bus.on('servers subscribe', serversSubscribeEvent);
        bus.on('servers unsubscribe', serversUnsubscribeEvent);
        bus.on('command send', commandSendEvent);
        bus.on('mods sync subscribe', modsSyncSubscribeEvent);
        bus.on('mods sync unsubscribe', modsSyncUnsubscribeEvent);
    }

    function unregisterEventEmitter() {
        bus.off('log subscribe', logSubscribeEvent);
        bus.off('log unsubscribe', logUnsubscribeEvent);
        bus.off('server status subscribe', serverStatusSubscribeEvent);
        bus.off('server version subscribe', serverVersionSubscribeEvent);
        bus.off('servers subscribe', serversSubscribeEvent);
        bus.off('servers unsubscribe', serversUnsubscribeEvent);
        bus.off('command send', commandSendEvent);
        bus.off('mods sync subscribe', modsSyncSubscribeEvent);
        bus.off('mods sync unsubscribe', modsSyncUnsubscribeEvent);
    }

    socket.onmessage = e => {
        const {room_name, message} = JSON.parse(e.data);
        bus.emit(room_name, message);
    }

    socket.onerror = e => {
        socket.close();
    }

    socket.onclose = e => {
        unregisterEventEmitter()
        // reconnect after 5 seconds
        setTimeout(connect, 5000);
    }

    socket.onopen = e => {
        registerEventEmitter(socket)
    }
}

connect();

export default bus;
