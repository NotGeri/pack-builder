/** @type {import('tailwindcss').Config} */
export default {
    content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
    theme: {
        extend: {
            colors: {
                transparent: 'transparent',
                current: 'currentColor',
                muted: '#949BA4',
                background: '#36393f',
                darker: '#2f3136',
                darkest: '#202225'
            }
        },
    },
    plugins: [],
};

