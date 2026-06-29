import EventEmitter from "events";

const bus = new EventEmitter();

const ws_scheme = window.location.protocol === "https:" ? "wss" : "ws";

function connect() {
    const socket = new WebSocket(ws_scheme + "://" + window.location.host + "/ws");

    function logSubscribeEvent() {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "subscribe",
                        value: "gamelog"
                    }
                }
            )
        );
    }

    function logUnsubscribeEvent() {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "unsubscribe",
                        value: "gamelog"
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

    function commandSendEvent(command) {
        socket.send(
            JSON.stringify(
                {
                    room_name: "",
                    controls: {
                        type: "command",
                        value: command
                    }
                }
            )
        );
    }

    function modsSyncSubscribeEvent() {
        socket.send(JSON.stringify({room_name: "", controls: {type: "subscribe", value: "mods_sync"}}));
    }

    function modsSyncUnsubscribeEvent() {
        socket.send(JSON.stringify({room_name: "", controls: {type: "unsubscribe", value: "mods_sync"}}));
    }

    function registerEventEmitter() {
        bus.on('log subscribe', logSubscribeEvent);
        bus.on('log unsubscribe', logUnsubscribeEvent);
        bus.on('server status subscribe', serverStatusSubscribeEvent);
        bus.on('server version subscribe', serverVersionSubscribeEvent);
        bus.on('command send', commandSendEvent);
        bus.on('mods sync subscribe', modsSyncSubscribeEvent);
        bus.on('mods sync unsubscribe', modsSyncUnsubscribeEvent);
    }

    function unregisterEventEmitter() {
        bus.off('log subscribe', logSubscribeEvent);
        bus.off('log unsubscribe', logUnsubscribeEvent);
        bus.off('server status subscribe', serverStatusSubscribeEvent);
        bus.off('server version subscribe', serverVersionSubscribeEvent);
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