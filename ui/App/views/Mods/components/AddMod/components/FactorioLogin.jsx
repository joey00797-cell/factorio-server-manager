import React, {useState} from "react";
import {useForm} from "react-hook-form";
import Input from "../../../../../components/Input";
import Label from "../../../../../components/Label";
import Button from "../../../../../components/Button";
import modsResource from "../../../../../../api/resources/mods";

const FactorioLogin = ({setIsFactorioAuthenticated}) => {

    const {register, handleSubmit} = useForm();
    const [isLoading, setIsLoading] = useState(false);

    const login = async ({username, token}) => {
        setIsLoading(true);
        setIsFactorioAuthenticated(false);

        try {
            await modsResource.portal.login(username, token);
            const isAuthenticated = await modsResource.portal.status();
            setIsFactorioAuthenticated(isAuthenticated);

            if (!isAuthenticated) {
                window.flash("Factorio accepted the login request, but the saved credentials could not be validated.", "red");
            }
        } catch (err) {
            window.flash(err.response?.data || "Factorio rejected this username/password or username/token.", "red");
        } finally {
            setIsLoading(false);
        }
    }

    return (
        <form onSubmit={handleSubmit(login)}>
            <div className="flex mb-4">
                <div className="w-1/2 mr-2">
                    <Label text="Username" htmlFor="username"/>
                    <Input register={register('username',{required: true})}/>
                </div>
                <div className="w-1/2 ml-2">
                    <Label text="Password or Token" htmlFor="password"/>
                    <Input type="password" register={register('token',{required: true})}/>
                </div>
            </div>
            <Button isSubmit={true} isLoading={isLoading}>Login</Button>
        </form>
    )
}

export default FactorioLogin;
