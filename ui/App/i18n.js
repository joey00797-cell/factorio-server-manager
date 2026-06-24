import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import Backend from 'i18next-http-backend';
import LanguageDetector from "i18next-browser-languagedetector";

i18n
    .use(Backend)
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
        fallbackLng: "en",
        backend: {
            loadPath: "/api/locales/{{lng}}",
        },
        detection: {
            order: ["localStorage", "cookie", "navigator"],
            cache: ["localStorage", "cookie"]
        },
        interpolation: {
            escapeValue: false
        },
        react: {
            useSuspense: false,
        },
        initImmediate: false,
    });

export default i18n;
