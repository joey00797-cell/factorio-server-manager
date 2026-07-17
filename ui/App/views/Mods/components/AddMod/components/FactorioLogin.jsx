import React, {useState} from "react";
import {useForm} from "react-hook-form";
import Input from "../../../../../components/Input";
import Label from "../../../../../components/Label";
import Button from "../../../../../components/Button";
import modsResource from "../../../../../../api/resources/mods";
import { useTranslation } from "react-i18next";

const FactorioLogin = ({setIsFactorioAuthenticated, setPortalUsername}) => {
    const { t } = useTranslation();
    const {register, handleSubmit} = useForm();
    const [isLoading, setIsLoading] = useState(false);

    const login = async ({username, token}) => {
        setIsLoading(true);
        modsResource.portal.login(username, token)
            .then(res => {
                setIsFactorioAuthenticated(true);
                if (setPortalUsername) setPortalUsername(username);
            })
            .catch(() => window.flash(t("login.factorio_login_error_message"), "red"))
            .finally(() => setIsLoading(false));
    }

    return (
        <form onSubmit={handleSubmit(login)}>
            <div className="flex items-center gap-2">
                <input
                    className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                    placeholder={t("username")}
                    {...register('username', {required: true})}
                />
                <input
                    type="password"
                    className="shadow appearance-none border py-2 px-3 text-black flex-1 min-w-0"
                    placeholder={t("login.password", "Password")}
                    {...register('token', {required: true})}
                />
                <Button isSubmit={true} isLoading={isLoading}>{t("login.login")}</Button>
            </div>
        </form>
    )
}

export default FactorioLogin;
