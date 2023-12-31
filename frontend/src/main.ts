import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import { createPinia } from 'pinia';
import main from './app.vue';

// Import Tailwind styles
import '../assets/main.css';

// Create the app
const app = createApp(main);

// Use a simple router
app.use(createRouter({
    history: createWebHistory('/'),
    routes: [ {
        path: '/',
        name: 'app',
        component: main
    } ]
}));

// Use global state management
app.use(createPinia());

// Mount the app
app.mount('#app');
