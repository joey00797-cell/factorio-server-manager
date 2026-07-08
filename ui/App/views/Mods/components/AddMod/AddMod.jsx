import React, {useEffect, useState} from "react";
import AddModForm from "./components/AddModForm";
import FactorioLogin from "./components/FactorioLogin";
import modResource from "../../../../../api/resources/mods";
import { useTranslation } from "react-i18next";



const AddMod = ({refetchInstalledMods, fuse, serverId}) => {
    const { t } = useTranslation();

    const [isFactorioAuthenticated, setIsFactorioAuthenticated] = useState(false);
    const [portalUsername, setPortalUsername] = useState("");

    useEffect(() => {
        (async () => {
            const status = await modResource.portal.status();
            if (status && status.logged_in) {
                setIsFactorioAuthenticated(true);
                setPortalUsername(status.username || "");
            }
        })();
    }, []);

    return isFactorioAuthenticated
        ? <AddModForm fuse={fuse} setIsFactorioAuthenticated={setIsFactorioAuthenticated} refetchInstalledMods={refetchInstalledMods} serverId={serverId} portalUsername={portalUsername}/>
        : <FactorioLogin setIsFactorioAuthenticated={setIsFactorioAuthenticated} setPortalUsername={setPortalUsername}/>
}

export default AddMod;
