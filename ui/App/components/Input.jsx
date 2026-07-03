import React from "react";

const Input = ({
                   register,
                   name,
                   placeholder = undefined,
                   type = "text",
                   defaultValue = undefined,
                   hasAutoComplete = true,
                   onKeyDown = () => undefined,
                   min = undefined,
                   max = undefined,
                   value = undefined,
                   disabled = false,
                   onChange = undefined
               }) => {
    const registeredProps = name && register ? register(name) : (register || {});

    return (
        <input
            className="shadow appearance-none border w-full py-2 px-3 text-black"
            placeholder={placeholder}
            {...registeredProps}
            type={type}
            onKeyDown={onKeyDown}
            autoComplete={hasAutoComplete ? "on" : "off"}
            defaultValue={defaultValue}
            min={min}
            max={max}
            value={value}
            disabled={disabled}
            {...(onChange ? {onChange} : {})}
        />
    )
}

export default Input;
