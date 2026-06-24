import React, {useState} from "react";
import {useForm} from "react-hook-form";
import Input from "../../../../../components/Input";
import Label from "../../../../../components/Label";
import Button from "../../../../../components/Button";
import modsResource from "../../../../../../api/resources/mods";
import { useTranslation } from "react-i18next";

const FactorioLogin = ({setIsFactorioAuthenticated}) => {
    const { t } = useTranslation();
    const {register, handleSubmit} = useForm();
    const [isLoading, setIsLoading] = useState(false);

    const login = ({username, token}) => {
        setIsLoading(true);
        modsResource.portal.login(username, token)
            .then(res => {
                setIsFactorioAuthenticated(true)
            })
            .catch(() => window.flash(t("login.factorio_login_error_message"), "red"))
            .finally(() => setIsLoading(false));
    }

    return (
        <form onSubmit={handleSubmit(login)}>
            <div className="flex mb-4">
                <div className="w-1/2 mr-2">
                    <Label text={t("username")} htmlFor="username"/>
                    <Input register={register('username',{required: true})}/>
                </div>
                <div className="w-1/2 ml-2">
                    <Label text={t("mods.add_mod.version")} htmlFor="password"/>
                    <Input type="password" register={register('token',{required: true})}/>
                </div>
            </div>
            <Button isSubmit={true} isLoading={isLoading}>{t("login.login")}</Button>
        </form>
    )
}

export default FactorioLogin;
