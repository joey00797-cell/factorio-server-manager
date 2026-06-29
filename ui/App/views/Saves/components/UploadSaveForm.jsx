import Button from "../../../components/Button";
import React, {useState} from "react";
import {useForm} from "react-hook-form";
import saves from "../../../../api/resources/saves";
import Error from "../../../components/Error";
import Label from "../../../components/Label";
import { useTranslation } from "react-i18next";


const UploadSaveForm = ({onSuccess, serverId}) => {
    const { t } = useTranslation();
    const [fileName, setFileName] = useState('');
    React.useEffect(() => setFileName(t('saves.upload_form.select_file')), [t]);
    const {register, handleSubmit, formState: {errors}} = useForm();

    const onSubmit = (data, e) => {
        saves.upload(data.savefile[0], serverId).then(_ => {
            e.target.reset();
            onSuccess();
        })
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="mb-6">
                <Label text={t("saves.upload_form.file_name")} htmlFor="savefile"/>
                <div className="relative bg-white shadow text-black w-full">
                    <input
                        className="absolute left-0 top-0 opacity-0 cursor-pointer w-full h-full"
                        {...register('savefile', {required: true})}
                        onChange={e => setFileName(e.currentTarget.files[0].name)}
                        type="file"/>
                    <div className="px-2 py-2">{fileName}</div>
                </div>
                <Error error={errors.savefile} message={t("saves.upload_form.save_file_error_message")}/>
            </div>
            <Button type="success" isSubmit={true}>{t("saves.upload_form.upload")}</Button>
        </form>
    )
}

export default UploadSaveForm;
