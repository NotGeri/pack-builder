<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import type { LinkState, Platform, Request } from '@/helpers';
import { config, useApi, useStore } from '@/helpers';

const api = useApi();
const store = useStore();
const request = reactive<Request>({});
const error = ref<string | undefined>('');

const validate = () => {
    let validLinks = 0;
    for (const link of request.links?.split('\n') ?? []) {
        if (link?.startsWith('http')) validLinks++;
    }

    if (validLinks == 0) {
        error.value = 'No links provided';
    } else {
        error.value = undefined;
    }
};

const selectPlatform = (id: string, options?: { platformVersion?: string, gameVersion?: string, links?: string }) => {

    // Set the ID
    request.platform = id;

    // Handle platforms that do not have platform versioning
    switch (id) {
        case 'spigot': {
            request.platform_version = 'latest';
            break;
        }
        default: {
            request.platform_version = options?.platformVersion;
            break;
        }
    }

    // Clear previous values
    request.game_version = options?.gameVersion;
    request.links = options?.links;
};

const platform = computed<Platform | null>(() => {
    if (!store.info) return null;
    if (!request.platform) return null;
    return store.info?.platforms[request.platform];
});

const submit = async () => {

    // Todo (notgeri): user friendlify
    const response = await api.newSessionRequest(request);
    if (!response.success) {
        console.error(response.error);
        return;
    }
    if (!response.raw?.ok || !response.data) {
        console.error(response);
        return;
    }

    // Initialize the link states
    const links: Record<string, LinkState> = {};
    for (const [ id, link ] of Object.entries(response.data.links)) {
        links[id] = { id, link };
    }

    // Update the data
    store.updateSession({
        id: response.data.id,
        links: links
    }, { fullSession: true });

    // Open the socket
    await api.openSocket();
};
</script>


<template>
    <!--  // Todo (notgeri):  -->
    <button @click="request.links = config.test; error = undefined">test</button>

    <div class="flex flex-col gap-3 justify-center items-center">

        <p class="text-xs text-muted">Select the platform:</p>

        <p v-if="!store.info" class="text-blue-400 text-xs">Loading platforms...</p>
        <div v-else class="flex flex-row justify-center items-center gap-3">
            <template v-for="[id, platform] of Object.entries(store.info.platforms)">
                <div @click="selectPlatform(id)"
                     :class="['flex flex-col justify-center items-center gap-1 cursor-pointer transition-colors grayscale text-muted hover:grayscale-0 hover:text-white', {'grayscale-0 text-white': request.platform == id}]">
                    <img :alt="`${platform.name}'s logo`"
                         :src="`/platforms/${id}.png`"
                         class="w-32 h-32 text-xs rounded border-2 border-darkest"
                         draggable="false"/>
                    <p>{{ platform.name }}</p>
                </div>
            </template>
        </div>

        <div v-if="request.platform" class="flex flex-col gap-3 justify-center items-center mt-5">
            <p class="text-xs text-muted">Select the game version:</p>

            <div class="flex flex-row gap-5">
                <template v-for="version of platform?.game_versions ?? []">
                    <button :class="['btn', {'primary': request.game_version == version}]"
                            @click="request.game_version = version">
                        {{ version }}
                    </button>
                </template>
            </div>
        </div>

        <div
            v-if="request.platform && request.game_version && platform?.platform_versions"
            class="flex flex-col gap-3 justify-center items-center mt-5">
            <p class="text-xs text-muted">Select the platform version:</p>

            <div class="flex flex-row gap-5">
                <template v-for="version of platform?.platform_versions ?? []">
                    <button :class="['btn', {'primary': request.platform_version == version}]"
                            @click="request.platform_version = version">
                        {{ version }}
                    </button>
                </template>
            </div>
        </div>

        <div v-if="request.platform && request.game_version && request.platform_version"
             class="flex flex-col justify-center items-center gap-3 mt-5 w-1/2">
            <p class="text-xs text-muted">Paste the list of links to download:</p>

            <textarea v-model="request.links" @input="validate" class="w-full" rows="15"/>

            <p class="text-xs text-red-400">{{ error }}</p>

            <button @click="submit" :disabled="!!store.session.id || error !== undefined" class="btn success">
                Send
            </button>
        </div>
    </div>
</template>
