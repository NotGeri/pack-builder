<script setup lang="ts">
import { onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { sleep, useApi, useStore, version } from '@/helpers';
import Link from '@/components/link.vue';
import Request from '@/components/request.vue';
import Report from '@/components/report.vue';
import Packages from '@/components/packages.vue';
import Toolbar from '@/components/toolbar.vue';

const store = useStore();
const api = useApi();
const route = useRoute();
const router = useRouter();

/**
 * Ensure the selected session ID stays
 * in-sync with the URL query
 */
watch(() => store.session.id, async (newId) => {
    await router.isReady();
    if (newId) {
        await router.replace({ query: { session: newId } });
    } else {
        const query = { ...route.query };
        delete (query.session);
        await router.replace({ query });
    }
});

/**
 * When a page load, check if we have a session ID
 * and if so, attempt to connect
 */
onMounted(async () => {
    let attempts = 0;
    let info = await api.getInfo();

    // Todo (notgeri): user friendlify
    while (!info.success || !info.raw?.ok) {
        await sleep(1000);
        info = await api.getInfo();
        if (attempts > 100) {
            console.error('Backend did not connect after 100 attempts, giving up!');
            return;
        }
    }

    // Successfully received data, we'll store it
    store.info = info.data;

    // See if we have a session query parameter
    await router.isReady();
    const id = route.query.session?.toString();
    if (!id) return;

    // Get the current session
    const session = await api.getSession(id);
    if (!session.success) return;


    // If it was found, update the data and open a socket
    if (session.raw?.ok && session.data) {
        store.updateSession(session.data, { fullSession: true });
        await api.openSocket();
    } else { // If not, we will just redirect, maybe send a message in the future
        if (session.raw?.status == 404) store.clearSession();
    }
});

/**
 * Start blank session
 */
const clearSession = () => {
    api.closeSocket();
    store.updateSession({ id: null, links: {}, overall_state: undefined, packages: {} }, { fullSession: true });
};
</script>

<template>
    <h1 class="text-3xl text-center mb-3">Pack Builder v{{ version }} ✨</h1>

    <div v-if="store.session.id" class="flex flex-row gap-2 justify-center items-center">
        <p>Session: {{ store.session.id ?? '-' }}</p>
        <button v-if="store.session.id" @click="clearSession" class="italic text-blue-300 inline">New?</button>
    </div>
    <Request v-else/>

    <div class="flex flex-col gap-3 mt-10">
        <template v-for="[id, state] of Object.entries(store.session.links)" :key="id">
            <Link v-bind="state"/>
        </template>
    </div>

    <Toolbar/>
    <Packages/>
    <Report v-if="store.session?.overall_state?.post_processing"/>
</template>
