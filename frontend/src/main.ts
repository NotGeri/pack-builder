import '../assets/main.css';
import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import app from './app.vue';

const router = createRouter({
    history: createWebHistory('/'),
    routes: [ {
        path: '/',
        name: 'app',
        component: app
    } ]
});

createApp(app).use(router).mount('#app');
