import React from "react";

const Checkbox = ({text, register, name, checked}) => {
    return (
        <label className="block text-gray-500 font-bold">
            <input
                className="mr-2 leading-tight"
                {...register(name)}
                type="checkbox" defaultChecked={checked}/>
            <span className="text-sm">{text}</span>
        </label>
    )
}

export default Checkbox;
