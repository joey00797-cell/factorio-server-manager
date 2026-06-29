import React, {useCallback, useState} from 'react';

import user from "../api/resources/user";
import Login from "./views/Login";
import {Navigate, Route, Routes} from "react-router";
import Controls from "./views/Controls";
import {BrowserRouter, Outlet} from "react-router-dom";
import Logs from "./views/Logs";
import FsmLogs from "./views/FsmLogs";
import Saves from "./views/Saves/Saves";
import Layout from "./components/Layout";
import Mods from "./views/Mods/Mods";
import UserManagement from "./views/UserManagement/UserManagment";
import ServerSettings from "./views/ServerSettings";
import GameSettings from "./views/GameSettings";
import ModOptions from "./views/ModOptions";
import Console from "./views/Console";
import Help from "./views/Help";
import "./i18n";


const App = () => {

    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const handleAuthenticationStatus = useCallback(async (status) => {
        if (status?.username) {
            setIsAuthenticated(true);
        }
    },[]);

    const handleLogout = useCallback(async () => {
        const loggedOut = await user.logout();
        if (loggedOut) {
            setIsAuthenticated(false);
        }
    }, []);

    const ProtectedRoute = ({isAuthenticated}) => {
        if (!isAuthenticated) {
            return <Navigate to="/login" state={{from: window.location.pathname}} />;
        }
        return <Outlet/>;
    }

    return (
        <BrowserRouter>
            <Routes>
                <Route path="login" element={<Login handleLogin={handleAuthenticationStatus}/>}/>

                {/* route with only `element` will cause the proper children to be place in `<Outlet/>` */}
                <Route element={<ProtectedRoute isAuthenticated={isAuthenticated}/> }>
                    <Route element={<Layout handleLogout={handleLogout} />}>
                        <Route index element={<Controls/>}/>
                        <Route path="saves" element={<Navigate to="/servers/1/saves" replace/>}/>
                        <Route path="mods" element={<Navigate to="/servers/1/mods" replace/>}/>
                        <Route path="server-settings" element={<Navigate to="/servers/1/server-settings" replace/>}/>
                        <Route path="game-settings" element={<Navigate to="/servers/1/game-settings" replace/>}/>
                        <Route path="mod-options" element={<Navigate to="/servers/1/mod-options" replace/>}/>
                        <Route path="console" element={<Navigate to="/servers/1/console" replace/>}/>
                        <Route path="logs" element={<Navigate to="/servers/1/logs" replace/>}/>
                        <Route path="servers/:serverId/saves" element={<Saves/>}/>
                        <Route path="servers/:serverId/mods" element={<Mods/>}/>
                        <Route path="servers/:serverId/server-settings" element={<ServerSettings/>}/>
                        <Route path="servers/:serverId/game-settings" element={<GameSettings/>}/>
                        <Route path="servers/:serverId/mod-options" element={<ModOptions/>}/>
                        <Route path="servers/:serverId/console" element={<Console/>}/>
                        <Route path="servers/:serverId/logs" element={<Logs/>}/>
                        <Route path="fsm-logs" element={<FsmLogs/>}/>
                        <Route path="user-management" element={<UserManagement/>}/>
                        <Route path="help" element={<Help/>}/>
                    </Route>
                </Route>
            </Routes>
        </BrowserRouter>
    );
}

export default App;
