import React, { useState, useEffect, useRef } from 'react';
import Modal from "./Modal";
import Button from "./Button";
import { useTranslation } from "react-i18next";

function ChangeLangDialog({isOpen, close, onSuccess}) {
    const { t, i18n } = useTranslation();
    const [langs, setLangs] = useState([]);
    const [uploading, setUploading] = useState(false);
    const [uploadMsg, setUploadMsg] = useState("");
    const [uploadMsgColor, setUploadMsgColor] = useState("green");
    const [pendingLang, setPendingLang] = useState(null); // после загрузки — предлагаем применить
    const [confirmOverwrite, setConfirmOverwrite] = useState(null); // имя файла для перезаписи
    const [pendingFile, setPendingFile] = useState(null); // файл ожидающий подтверждения
    const fileRef = useRef();

    const fetchLangs = () => {
        fetch("/api/locales/list")
            .then(r => r.json())
            .then(data => setLangs(data))
            .catch(() => setLangs([]));
    };

    useEffect(() => {
        if (isOpen) fetchLangs();
    }, [isOpen]);

    const changeLang = (code) => {
        i18n.changeLanguage(code);
        close();
    };

    const downloadLocale = (lang) => {
        const a = document.createElement("a");
        a.href = `/api/locales/${lang}`;
        a.download = `${lang}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    };

    const doUpload = async (file) => {
        setUploading(true);
        setUploadMsg("");
        setPendingLang(null);
        const form = new FormData();
        form.append("locale", file);
        try {
            const r = await fetch("/api/locales/upload", { method: "POST", body: form });
            if (r.ok) {
                // Определяем код языка из имени файла
                const code = file.name.replace('.json', '');
                fetchLangs();
                setUploadMsgColor("green");
                setUploadMsg(t("lang_upload_ok"));
                setPendingLang(code);
            } else {
                const text = await r.text();
                setUploadMsgColor("red");
                setUploadMsg(t("lang_upload_err") + ": " + text);
            }
        } catch {
            setUploadMsgColor("red");
            setUploadMsg(t("lang_upload_err"));
        }
        setUploading(false);
        if (fileRef.current) fileRef.current.value = "";
        setPendingFile(null);
        setConfirmOverwrite(null);
    };

    const handleFileChange = async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        // Проверяем дубликат
        const code = file.name.replace('.json', '');
        const exists = langs.find(l => l.code === code);
        if (exists) {
            setPendingFile(file);
            setConfirmOverwrite(file.name);
            return;
        }
        await doUpload(file);
    };

    return (
        <Modal
            title={t("lang")}
            content={
                <>
                    {/* Список языков */}
                    {langs.map(l => (
                        <div key={l.code} className="flex items-center mb-2 gap-2">
                            <Button
                                className="flex-1"
                                type={i18n.language === l.code ? "success" : "default"}
                                onClick={() => changeLang(l.code)}
                            >
                                {l.name} {!l.builtin && "🌐"}
                                {i18n.language === l.code && " ✓"}
                            </Button>
                            <Button
                                size="sm"
                                title={t("lang_download_current")}
                                onClick={() => downloadLocale(l.code)}
                            >
                                ⬇
                            </Button>
                        </div>
                    ))}

                    {/* Сообщение */}
                    {uploadMsg && (
                        <p className={`text-sm mt-2 text-center text-${uploadMsgColor}-400`}>
                            {uploadMsg}
                        </p>
                    )}

                    {/* Предложение применить загруженный язык */}
                    {pendingLang && (
                        <div className="mt-3 p-2 border border-green-400 rounded text-sm">
                            <p className="mb-2 text-green-400">{t("lang_apply_question")}</p>
                            <div className="flex gap-2 flex-wrap">
                                <Button size="sm" type="success" onClick={() => {
                                    i18n.changeLanguage(pendingLang);
                                    setPendingLang(null);
                                    window.location.reload();
                                }}>{t("confirm")}</Button>
                                <Button size="sm" type="danger" onClick={() => setPendingLang(null)}>{t("cancel")}</Button>
                            </div>
                        </div>
                    )}

                    {/* Подтверждение перезаписи */}
                    {confirmOverwrite && (
                        <div className="mt-3 p-2 border border-orange rounded text-sm">
                            <p className="mb-2 text-orange-400">{t("lang_overwrite_question", `Файл ${confirmOverwrite} уже существует. Заменить?`)}</p>
                            <div className="flex gap-2 flex-wrap">
                                <Button size="sm" type="success" onClick={() => doUpload(pendingFile)}>{t("confirm")}</Button>
                                <Button size="sm" type="danger" onClick={() => {
                                    setConfirmOverwrite(null);
                                    setPendingFile(null);
                                    if (fileRef.current) fileRef.current.value = "";
                                }}>{t("cancel")}</Button>
                            </div>
                        </div>
                    )}

                    <input
                        ref={fileRef}
                        type="file"
                        accept=".json"
                        className="hidden"
                        onChange={handleFileChange}
                    />
                </>
            }
            actions={
                <div className="flex flex-wrap justify-between w-full gap-2">
                    <div className="flex gap-2 flex-wrap">
                        <Button size="sm" onClick={() => downloadLocale("en")}>
                            ⬇ {t("lang_download_template")}
                        </Button>
                        <Button size="sm" onClick={() => fileRef.current.click()} disabled={uploading}>
                            ⬆ {uploading ? t("lang_uploading") : t("lang_upload")}
                        </Button>
                    </div>
                    <Button size="sm" type="danger" onClick={close}>
                        {t("cancel")}
                    </Button>
                </div>
            }
            isOpen={isOpen}
            onSuccess={onSuccess}
        />
    );
}

export default ChangeLangDialog;
