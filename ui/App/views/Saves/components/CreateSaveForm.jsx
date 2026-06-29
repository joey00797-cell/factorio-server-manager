import {useForm} from "react-hook-form";
import Button from "../../../components/Button";
import React, {useState} from "react";
import saves from "../../../../api/resources/saves";
import Label from "../../../components/Label";
import Input from "../../../components/Input";
import Error from "../../../components/Error";
import { useTranslation } from "react-i18next";

const CreateSaveForm = ({onSuccess, isFactorioInstalled = true}) => {
    const { t } = useTranslation();
    const {register, handleSubmit, formState: {errors}} = useForm();
    const [isLoading, setIsLoading] = useState(false);
    const [apiError, setApiError] = useState(null);

    const onSubmit = async (data, e) => {
        setIsLoading(true);
        setApiError(null);
        saves.create(data.savefile)
            .then(() => {
                e.target.reset();
                onSuccess();
            })
            .catch((err) => {
                const msg = err?.response?.data || "saves.create_error";
                setApiError(msg);
            })
            .finally(() => setIsLoading(false));
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            {!isFactorioInstalled && (
                <div className="mb-4 p-3 bg-red bg-opacity-20 border border-red rounded text-red-light font-bold">
                    ⚠ {t("saves.factorio_not_installed")}
                </div>
            )}
            <div className="mb-6">
                <Label text={t("saves.save_form.file_name")} htmlFor="savefile"/>
                <Input register={register('savefile', {required: true})} disabled={!isFactorioInstalled}/>
                <Error error={errors.savefile} message={t("saves.save_form.save_file_error_message")}/>
            </div>
            {apiError && (
                <div className="mb-4 p-3 bg-red bg-opacity-20 border border-red rounded text-red-light font-bold">
                    ⚠ {t(apiError, apiError)}
                </div>
            )}
            <Button type="success" isLoading={isLoading} isSubmit={true} isDisabled={!isFactorioInstalled}>{t("saves.save_form.create_save")}</Button>
        </form>
    )
}

export default CreateSaveForm;
