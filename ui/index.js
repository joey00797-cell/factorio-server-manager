import regeneratorRuntime from "regenerator-runtime"
import Bus from "./notifications"
import React, { Suspense, useState, useEffect } from 'react';
import ReactDOM from 'react-dom/client';
import App from './App/App.jsx';
import i18n from './App/i18n.js';

window.flash = (message, color="gray-light") => Bus.emit('flash', ({message, color}));

const Root = () => {
    const [ready, setReady] = useState(i18n.isInitialized);

    useEffect(() => {
        if (!i18n.isInitialized) {
            i18n.on('initialized', () => setReady(true));
        }
    }, []);

    if (!ready) return <div style={{background:'#1a1a1a', minHeight:'100vh'}}/>;
    return <App/>;
};

const root = ReactDOM.createRoot(document.getElementById('app'));
root.render(<Root/>);
