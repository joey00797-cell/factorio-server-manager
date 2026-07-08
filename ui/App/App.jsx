import React, {useCallback, useState} from 'react';

import user from "../api/resources/user";
import Login from "./views/Login";
import {Navigate, Route, Routes} from "react-router";
import Controls from "./views/Controls";
import {BrowserRouter, Outlet} from "react-router-dom";
import ServerLogs from "./views/ServerLogs";
import Saves from "./views/Saves/Saves";
import Layout from "./components/Layout";
import RedirectToServerScoped from "./components/RedirectToServerScoped";
import Mods from "./views/Mods/Mods";
import UserManagement from "./views/UserManagement/UserManagment";
import ServerSettings from "./views/ServerSettings";
import GameSettings from "./views/GameSettings";
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
                        <Route path="saves" element={<RedirectToServerScoped suffix="saves"/>}/>
                        <Route path="mods" element={<RedirectToServerScoped suffix="mods"/>}/>
                        <Route path="server-settings" element={<RedirectToServerScoped suffix="server-settings"/>}/>
                        <Route path="game-settings" element={<RedirectToServerScoped suffix="game-settings"/>}/>
                        <Route path="console" element={<RedirectToServerScoped suffix="logs"/>}/>
                        <Route path="logs" element={<RedirectToServerScoped suffix="logs"/>}/>
                        <Route path="servers/:serverId/saves" element={<Saves/>}/>
                        <Route path="servers/:serverId/mods" element={<Mods/>}/>
                        <Route path="servers/:serverId/server-settings" element={<ServerSettings/>}/>
                        <Route path="servers/:serverId/game-settings" element={<GameSettings/>}/>
                        <Route path="servers/:serverId/logs" element={<ServerLogs/>}/>
                        <Route path="user-management" element={<UserManagement/>}/>
                        <Route path="help" element={<Help/>}/>
                    </Route>
                </Route>
            </Routes>
        </BrowserRouter>
    );
}

export default App;
