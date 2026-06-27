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
                   disabled = false
               }) => {
    return (
        <input
            className="shadow appearance-none border w-full py-2 px-3 text-black"
            placeholder={placeholder}
            {...(name && register ? register(name) : (register || {}))}
            type={type}
            onKeyDown={onKeyDown}
            autoComplete={hasAutoComplete ? "on" : "off"}
            defaultValue={defaultValue}
            min={min}
            max={max}
            value={value}
            disabled={disabled}
        />
    )
}

export default Input;
