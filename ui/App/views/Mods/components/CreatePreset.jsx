import React, {useState} from "react";
import Button from "../../../components/Button";
import Modal from "../../../components/Modal";
import Label from "../../../components/Label";
import Input from "../../../components/Input";
import {useForm} from "react-hook-form";
import presetsResource from "../../../../api/resources/presets";
import { useTranslation } from "react-i18next";

const CreatePreset = ({serverId, onSuccess}) => {
    const { t } = useTranslation();
    const [isCreating, setIsCreating] = useState(false);
    const [isOpen, setIsOpen] = useState(false);
    const {handleSubmit, register, reset} = useForm();

    const createPreset = (data) => {
        setIsCreating(true);
        presetsResource.save(serverId, data.name, data.description || "")
            .then(onSuccess)
            .catch(() => window.flash(t("presets.save_error", "Failed to save preset"), "red"))
            .finally(() => {
                setIsCreating(false);
                setIsOpen(false);
                reset();
            });
    };

    return <>
        <Button size="sm" onClick={() => setIsOpen(true)}>
            {t("presets.add", "Add Preset")}
        </Button>
        <Modal title={t("presets.title", "Mod Presets")} isOpen={isOpen} content={
            <form onSubmit={handleSubmit(createPreset)}>
                <div className="mb-4">
                    <Label text={t("name")} htmlFor="name"/>
                    <Input register={register("name", {required: true})}/>
                </div>
                <div className="mb-4">
                    <Label text={t("presets.description", "Description")} htmlFor="description"/>
                    <Input register={register("description")}/>
                </div>
                <Button size="sm" isLoading={isCreating} isSubmit={true}>
                    {t("presets.save", "Save")}
                </Button>
            </form>
        }
        actions={
            <Button onClick={() => setIsOpen(false)} size="sm" type="danger">
                {t("cancel", "Cancel")}
            </Button>
        }
        />
    </>
};

export default CreatePreset;
