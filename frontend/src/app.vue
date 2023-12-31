<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { v4 as newUuid } from 'uuid';
import config from '@/config';
import { errors, messages, version } from '@/const';
import Link from '@/link.vue';

type Status = 'success' | 'warning' | 'error'

type Simple = {
    status?: Status
    message?: string
    data?: any
}

type Version = {
    id: string
    link: string
    is_external: boolean
    url: string
    platforms: string[]
    game_versions: string[]
}

type PluginType = 'spigot' | 'modrinth'

export type PluginInfo = {
    type: PluginType
    id: string
    link: string
    name: string
    description: string
    contributors: string
    premium: boolean
    versions: Version[]
    icon_link: string
};

type Preliminary = {
    status: Status
    message: string
    error: string
    failed_attempts: Record<string, Record<string, string>>
    plugin_info?: PluginInfo
    links: Record<string, boolean>
    certain: boolean
};

type Download = {
    status: Status
    message: string
    path: string
    size: number
};

type Dependency = {
    name: string
    other_plugin: boolean
    search?: Preliminary
    download?: Download
};

type PostProcessing = {
    dependencies?: Dependency[]
};

export type LinkState = {
    id: string
    link: string
    preliminary?: Preliminary
    download?: Download
    post_processing?: PostProcessing
};

type Package = {
    status: Status
    downloadable: boolean
    message: string
    name: string
    type: 'client' | 'server' | 'misc'
    size: number
}

type Session = {
    id?: string | null
    links: Record<string, LinkState>
    packages?: Record<string, Package>
    overall_state?: {
        initialized: boolean
        preliminary: boolean
        download: boolean
        post_processing: boolean
        deleted: boolean
    }
};

type LinkReport = {};

type Report = {
    links: Record<string, LinkReport>
    toClient?: string
};

type VerifiedWebSocket = WebSocket & {
    verified: boolean
}

type Platform = {
    name: string
    platform_versions: string[]
    game_versions: string[]
}

type Info = {
    platforms: Record<string, Platform>
}

type Request = {
    links?: string
    platform?: string
    platform_version?: string
    game_version?: string
}

type SessionData = Session & {
    // If we are reconnecting to a session, the server has a map,
    // so we'll need to recover the links text area manually
    request?: Request & {
        links?: Record<string, string>
    }
}

const info = ref<Info | undefined>();
const request = reactive<Request>({});
const session = reactive<Session>({ links: {} });
const report = ref<Report | null>(null);

const socket = ref<VerifiedWebSocket | null>(null);
let openSocketRequest: Promise<boolean> | null = null;

const route = useRoute();
const router = useRouter();

/**
 * Send a new API request
 * @param endpoint The relative endpoint
 * @param method The HTTP method
 * @param data Optional data to pass
 */
const fetchApi = (endpoint: string, method: 'GET' | 'POST' | 'DELETE' | 'PATCH' | 'HEAD' = 'GET', data: any = null) => {
    const options: RequestInit = {
        method,
        credentials: 'include',
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json',
        },
    };

    if (method !== 'GET' && data) options.body = JSON.stringify(data);
    return fetch(`${config.backend.ssl ? 'https' : 'http'}://${config.backend.endpoint}/api${endpoint}`, options);
};

/**
 * Ensure the selected session ID stays
 * in-sync with the URL query
 */
watch(() => session.id, async (newId) => {
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
    // Fetch all the information about the checker, such as versions
    fetchApi('/info').then(async response => {
        if (!response.ok) return;
        info.value = await response.json(); // Todo (notgeri): redo these with transformers where we explicitly define what keys to add, so we somewhat verify the data
    });

    await router.isReady();
    const id = route.query.session;
    if (!id) return;

    const response = await fetchApi(`/sessions/${id}`);
    if (!response.ok) {
        // If the session is not found, we will just redirect;
        // maybe send a message in the future
        if (response.status == 404) {
            session.id = null;
        }
        return;
    }

    // Update the data
    const data = await response.json();
    updateSession(data, { fullSession: true, request: true });

    // Open the socket
    await openSocket();
});

/**
 * Update our current session with fresh data
 * @param data The data to update it with
 * @param options A list of options
 */
const updateSession = (data: SessionData, options?: { request?: boolean, fullSession?: boolean }) => {
    session.packages = data.packages;
    session.overall_state = data.overall_state;

    if (options?.fullSession) {
        session.id = data.id;
        session.links = data.links;
    }

    if (options?.request && data.request?.platform) {

        // Recover the links text area
        let links;
        if (data.request.links) {
            links = '';
            for (const link of Object.values(data.request.links)) {
                links += `${link}\n`;
            }
            request.links = links;
        }

        selectPlatform(data.request.platform, {
            platformVersion: data.request.platform_version,
            gameVersion: data.request.game_version,
            links,
        });
    }
};

/**
 * Handle messages from the server
 * @param message The main message or command that was sent
 * @param data Any additional data or null
 */
const handleMessage = (message: string, data: any = null) => {
    switch (message) {

        // We do not need to do anything for these
        case messages.PRELIMINARY_START:
        case messages.PROCESS_START:
        case messages.PACKAGE_START:
        case messages.GET_DOWNLOAD_START : {
            break;
        }

        // We will update each step when they are handled
        case messages.PRELIMINARY_STEP:
        case messages.PROCESS_STEP: {
            const state = data as LinkState;
            session.links[state.id] = state;
            break;
        }

        // Whenever a stage is done, we'll update the rest of the session data
        case messages.PRELIMINARY_DONE:
        case messages.PACKAGE_DONE:
        case messages.PROCESS_DONE:
        case messages.GET_DOWNLOAD_DONE: {
            updateSession(data as SessionData);
            break;
        }

        case messages.DELETED: {
            session.id = null;
            break;
        }

        default:
            console.error('Unable to handle command:', message, data);
            break;
    }
};

/**
 * Send a new message to the server
 * @param message The message or command to send
 * @param data The raw data to send or null if there is none
 */
const sendMessage = async (message: string, data: any = null) => {

    // If we don't have a socket, or it isn't yet verified, we'll open one first,
    // or if it's already in progress, await it
    if (!socket.value?.verified) {
        if (!await openSocket()) {
            console.error('Unable to send message, openSocket return false: ', message, data);
            return;
        }
    }

    // If the socket still isn't verified, there is nothing we can do
    if (!socket.value?.verified) {
        console.error('Unable to send message, socket not verified: ', message, data);
        return;
    }

    // Then we just serialize our message and send it
    let payload = `${message}`;
    if (data) payload += ` ${JSON.stringify(data)}`;
    socket.value.send(payload);
};

/**
 * Close the current websocket connection to free up resources
 */
const closeSocket = () => {
    if (!socket.value) return;
    try {
        socket.value.close();
        socket.value = null;
    } catch (e) {
        // Don't care, didn't ask
    }
};

/**
 * Open a new websocket for the session
 * This uses a global promise, so in case several consumers
 * call asynchronously, it should still return a single promise
 */
const openSocket = (): Promise<boolean> => {

    // Return if there is an existing running promise
    if (openSocketRequest !== null) return openSocketRequest;

    openSocketRequest = new Promise<boolean>(async resolve => {
        if (socket.value && socket.value.verified) return resolve(true);
        if (!session.id) return resolve(false);

        const newSocket = new WebSocket(`${config.backend.ssl ? 'wss' : 'ws'}://${config.backend.endpoint}/api/sessions/${session.id}/socket`) as VerifiedWebSocket;
        console.debug('New socket created');

        newSocket.onmessage = event => {
            if (event.type !== 'message') {
                console.error('Unable to handle event: ', event);
                return;
            }

            // Parse the raw message
            const raw = event.data.toString().split(' ');
            if (!raw || raw.length == 0) {
                console.error('Unable to handle event: ', event);
                return;
            }

            // The main command to execute for this session
            const message = raw[0];

            // The server will send a message if the websocket
            // connection was established successfully
            if (message == messages.CONNECTED) {
                newSocket.verified = true;
                resolve(true);
                return;
            }

            // Backend can pass some data as JSON
            raw.shift();
            const rawData = raw.join(' ');
            const data = rawData ? JSON.parse(rawData) : null;

            handleMessage(message, data);
        };

        newSocket.onerror = event => console.log('Socket error:', event);

        newSocket.onclose = () => {
            socket.value = null;
            if (resolve) resolve(false);
        };

        socket.value = newSocket;
    }).finally(() => {
        openSocketRequest = null;
    });

    return openSocketRequest;
};

const submit = async () => {
    if (session.id || !request.links) return;

    // Delete any previous links, mostly for testing
    for (const key of Object.keys(session.links)) delete (session.links[key]);

    // Reformat the request so the links all have a unique ID
    const formattedRequest: Omit<Request, 'links'> & {
        links: Record<string, string>
    } = { ...request, links: {} };

    for (const link of request.links.split('\n')) {
        if (!link || !link.startsWith('http')) continue;

        let id = newUuid();
        while (formattedRequest.links[id]) id = newUuid();

        formattedRequest.links[id] = link;
        session.links[id] = {
            id, link: link
        };
    }

    // Send the request
    const response = await fetchApi('/sessions', 'POST', formattedRequest);
    if (!response.ok) return;

    // Set our new session ID so we can send commands
    const data = await response.json();
    session.id = data.id;

    // Open the socket
    await openSocket();
};

/**
 * Handle downloading a package for our session
 * @param packageId The ID of the package
 * @param mode Whether to copy the link to the clipboard or download it directly
 */
const handleDownload = async (packageId: string, mode: 'copy' | 'download') => {
    if (!session) return;

    const link = `${config.backend.ssl ? 'https' : 'http'}://${config.backend.endpoint}/api/sessions/${session.id}/download/${packageId}`;

    switch (mode) {
        case 'copy': {
            if (!navigator.clipboard.writeText) {
                console.error('Unable to copy to clipboard: writeText unavailable');
                return;
            }

            await navigator.clipboard.writeText(link);
            break;
        }

        case 'download': {
            window.open(link, '_blank');
            break;
        }
    }
};

/**
 * Generate a report that the user can copy to clients
 */
const generateReport = () => {
    if (!session || !session.overall_state?.post_processing) return;

    const issues: Record<string, string[]> = {
        version: [],
        dependencies: []
    };

    for (const state of Object.values(session.links)) {
        // Collect dependencies that had to be downloaded
        if (state.post_processing?.dependencies && state.post_processing?.dependencies?.length > 0) {
            for (const dependency of state.post_processing.dependencies) {
                if (dependency.other_plugin) continue;
                if (dependency.download?.status !== 'success') continue;

                const name = dependency.search?.plugin_info?.name;
                if (!name) continue;
                if (issues.dependencies.includes(name)) continue;

                issues.dependencies.push(name);
            }
        }

        // Collect common errors that can't really be solved by support
        if (state.preliminary?.status !== 'error') continue;
        switch (state.preliminary.error) {
            case errors.NO_SUITABLE_VERSION: {
                issues.version.push(state.link);
                break;
            }
        }
    }

    let toClient = 'Dear [FIRSTNAME],\nYour installation is now complete!\n';

    if (issues.version.length > 0) {
        toClient += '\nThe following plugins are not compatible with your desired Minecraft version:\n';
        toClient += `${issues.version.map(link => `- ${link}`).join('\n')}\n`;
    }

    if (issues.dependencies.length > 0) {
        toClient += '\nThe following dependency plugins were also installed:\n';
        toClient += `${issues.dependencies.map(pluginName => `- ${pluginName}`).join('\n')}\n`;
    }

    report.value = {
        links: {},
        toClient,
    };
};

/**
 * Start blank session
 */
const clearSession = () => {
    closeSocket();
    updateSession({ id: null, links: {}, overall_state: undefined, packages: {} }, { fullSession: true });
    request.links = '';
};

const platform = computed<Platform | null>(() => {
    if (!info.value) return null;
    if (!request.platform) return null;
    return info.value.platforms[request.platform];
});

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

// Todo (notgeri):
const temp = computed(() => JSON.stringify(session, null, 4));
</script>

<template>
    <!-- Todo (notgeri): -->
    <div class="absolute top-1 left-1">
        <textarea :value="temp" class="w-1 h-1 resize-none"></textarea>
        <button @click="request.links = config.test">test</button>
    </div>

    <h1 class="text-3xl text-center mb-3">Pack Builder v{{ version }} ✨</h1>

    <div v-if="session.id" class="flex flex-col gap-3 justify-center items-center">
        <p>
            Session: {{ session.id ?? '-' }}
            <button v-if="session.id" @click="clearSession" class="italic inline text-blue-300">New?</button>
        </p>
        <h1 class="text-center text-4xl">
            <span v-if="socket" class="text-green-400">Online</span>
            <span v-else class="text-red-400">Offline</span>
        </h1>
    </div>

<!--  // Currently doing:  () add a button to show this section again, but before that break out into components already -->
    <div v-else class="flex flex-col gap-3 justify-center items-center">

        <p class="text-xs text-muted">Select the platform:</p>

        <p v-if="!info" class="text-blue-400 text-xs">Loading platforms...</p>
        <div v-else class="flex flex-row justify-center items-center gap-3">
            <template v-for="[id, platform] of Object.entries(info.platforms)">
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

            <textarea v-model="request.links" class="w-full" rows="15"/>

            <button @click="submit" :disabled="!!session.id || !request.links" class="btn success">
                Send
            </button>
        </div>
    </div>

    <div class="flex flex-col gap-3 mt-10">
        <template v-for="[id, state] of Object.entries(session.links)" :key="id">
            <Link v-bind="state" @toggleLink="data => sendMessage(messages.TOGGLE_LINK, data)"/>
        </template>
    </div>

    <div v-if="session.id"
         class="flex flex-row gap-3 mt-10 justify-center items-center fixed bottom-10 left-1/2 transform -translate-x-1/2 bg-black p-4 rounded-xl">
        <button class="btn success"
                @click="sendMessage(messages.PRELIMINARY)">
            Preliminary
        </button>
        <button class="btn success"
                :disabled="!session?.overall_state?.preliminary"
                @click="sendMessage(messages.PROCESS)">
            Process
        </button>
        <button class="btn success"
                :disabled="!session?.overall_state?.download"
                @click="sendMessage(messages.PACKAGE)">
            Package
        </button>
        <button class="btn primary"
                :disabled="!session?.overall_state?.post_processing"
                @click="generateReport">
            Report
        </button>
        <button class="btn danger"
                @click="sendMessage(messages.DELETE)">
            Delete
        </button>
    </div>

    <div v-if="session.packages" class="flex flex-row gap-3 justify-center items-center mt-3">
        <template v-for="[id, pkg] of Object.entries(session.packages)" :key="id">
            <div class="flex flex-row gap-3">
                <p class="text-red-400" v-if="pkg.status == 'error'">{{ pkg.message }}</p>
                <button class="btn primary"
                        @click="sendMessage(messages.GET_DOWNLOAD, id)"
                        @disabled="pkg.status != 'success'">
                    Request download for {{ pkg.name }}
                </button>

                <div v-if="pkg.downloadable" class="flex flex-row gap-3">
                    <button class="btn" @click="handleDownload(id, 'copy')">Copy to clipboard</button>
                    <button class="btn" @click="handleDownload(id, 'download')">Download to computer</button>
                </div>
            </div>
        </template>
    </div>

    <div v-if="report" class="mt-3">
        <textarea :value="report.toClient" rows="10"></textarea>
    </div>
</template>

