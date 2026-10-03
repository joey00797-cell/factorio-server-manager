import React, { useEffect, useRef, useState } from 'react';
import Bus from '../../notifications';

const DURATION = 4000;

export const Flash = () => {
    const [visibility, setVisibility] = useState(false);
    const [message, setMessage] = useState('');
    const [color, setColor] = useState('');
    const [progress, setProgress] = useState(100);
    const timerRef = useRef(null);
    const intervalRef = useRef(null);
    const hoveredRef = useRef(false);
    const startTimeRef = useRef(null);
    const remainingRef = useRef(DURATION);

    const clearTimers = () => {
        clearTimeout(timerRef.current);
        clearInterval(intervalRef.current);
    };

    const startTimer = (remaining) => {
        clearTimers();
        startTimeRef.current = Date.now();
        remainingRef.current = remaining;

        timerRef.current = setTimeout(() => {
            setVisibility(false);
            setProgress(100);
        }, remaining);

        intervalRef.current = setInterval(() => {
            const elapsed = Date.now() - startTimeRef.current;
            const pct = Math.max(0, 100 - (elapsed / remaining) * 100);
            setProgress(pct);
        }, 30);
    };

    const flashListener = ({message, color}) => {
        setVisibility(true);
        setMessage(message);
        setColor(color);
        setProgress(100);
        remainingRef.current = DURATION;
        if (!hoveredRef.current) startTimer(DURATION);
    };

    useEffect(() => {
        Bus.addListener('flash', flashListener);
        return () => {
            Bus.removeListener('flash', flashListener);
            clearTimers();
        };
    }, []);

    const handleMouseEnter = () => {
        hoveredRef.current = true;
        const elapsed = Date.now() - (startTimeRef.current || Date.now());
        remainingRef.current = Math.max(0, remainingRef.current - elapsed);
        clearTimers();
    };

    const handleMouseLeave = () => {
        hoveredRef.current = false;
        startTimer(remainingRef.current);
    };

    if (!visibility) return null;

    return (
        <div
            onClick={() => setVisibility(false)}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            className={`bg-${color} cursor-pointer accentuated rounded fixed bottom-0 right-0 mr-8 mb-8 px-4 py-2 z-50 max-w-1/2 lg:max-w-none`}
        >
            <p>{message}</p>
            <div className="mt-1 h-0.5 bg-black bg-opacity-20 rounded overflow-hidden">
                <div
                    className="h-full bg-white bg-opacity-60 transition-none"
                    style={{width: `${progress}%`}}
                />
            </div>
        </div>
    );
};

export default Flash;
