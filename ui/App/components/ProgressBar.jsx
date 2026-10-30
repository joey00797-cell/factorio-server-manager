import React from "react";

const ProgressBar = ({label, percent, detail}) => {
    const p = Math.max(0, Math.min(100, percent || 0));
    return (
        <div className="mb-2">
            <div className="flex justify-between text-sm mb-1"><span>{label}</span><span>{detail}</span></div>
            <div className="w-full bg-gray h-2"><div className="bg-orange h-2" style={{width: p + "%"}}/></div>
        </div>
    );
};

export default ProgressBar;
