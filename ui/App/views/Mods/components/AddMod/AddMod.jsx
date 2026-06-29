import React, {useEffect, useState} from "react";
import AddModForm from "./components/AddModForm";
import FactorioLogin from "./components/FactorioLogin";
import modResource from "../../../../../api/resources/mods";
import { useTranslation } from "react-i18next";



const AddMod = ({refetchInstalledMods, fuse, serverId}) => {
    const { t } = useTranslation();

    const [isFactorioAuthenticated, setIsFactorioAuthenticated] = useState(false);

    useEffect(() => {
        (async () => {
            setIsFactorioAuthenticated(await modResource.portal.status())
        })();
    }, []);

    return isFactorioAuthenticated
        ? <AddModForm fuse={fuse} setIsFactorioAuthenticated={setIsFactorioAuthenticated} refetchInstalledMods={refetchInstalledMods} serverId={serverId}/>
        : <FactorioLogin setIsFactorioAuthenticated={setIsFactorioAuthenticated}/>
}

export default AddMod;
