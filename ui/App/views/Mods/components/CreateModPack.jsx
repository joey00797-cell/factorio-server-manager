import React, {useState} from "react";
import Button from "../../../components/Button";
import Modal from "../../../components/Modal";
import Label from "../../../components/Label";
import Input from "../../../components/Input";
import {useForm} from "react-hook-form";
import modsResource from "../../../../api/resources/mods";
import { useTranslation } from "react-i18next";

const CreateModPack = ({onSuccess}) => {
    const { t } = useTranslation();
    const [isCreating, setIsCreating] = useState(false);
    const [isOpen, setIsOpen] = useState(false);

    const {handleSubmit, register} = useForm();

    const createModPack = (data) => {
        setIsCreating(true);

        modsResource.packs
            .create(data.name)
            .then(onSuccess)
            .finally(() => {
                setIsCreating(false)
                setIsOpen(false);
            });
    }

    return <>
        <Button size="sm" onClick={() => setIsOpen(true)}>{t("mods.add_modpack_with_current_mods")}</Button>
        <Modal title={t("mods.mod_packs")} isOpen={isOpen} content={
            <form onSubmit={handleSubmit(createModPack)}>
                <div className="mb-4">
                    <Label text={t("name")} htmlFor="name"/>
                    <Input register={register('name',{required: true})}/>
                </div>
                <Button size="sm" isLoading={isCreating} isSubmit={true}>Create</Button>
            </form>
        }
        actions={
            <Button onClick={() => setIsOpen(false)} size="sm" type="danger">Cancel</Button>
        }
        />
    </>
}

export default CreateModPack;
