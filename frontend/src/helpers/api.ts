import { defineStore } from 'pinia';
import { v4 as newUuid } from 'uuid';
import { config, messages, useStore } from '@/helpers';

// For API calls, if the fetch itself fails, we return a success: false and an optional error
// Otherwise, we'll try our best to parse the JSON as the given type
type ApiResponse<T> = {
    success: boolean
    data?: T
    error?: unknown
    raw?: Response
};

type Status = 'success' | 'warning' | 'error'

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

export type Package = {
    status: Status
    downloadable: boolean
    message: string
    name: string
    type: 'client' | 'server' | 'misc'
    size: number
}

export type Session = {
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

export type Platform = {
    name: string
    platform_versions: string[]
    game_versions: string[]
}

export type Info = {
    platforms: Record<string, Platform>
}

export type Request = {
    links?: string
    platform?: string
    platform_version?: string
    game_version?: string
}

// If we are reconnecting to a session, the server has a map,
// so we'll need to recover the links text area manually
export type SessionData = Session & {
    request?: Request & {
        links?: Record<string, string>
    }
}

type VerifiableWebSocket = WebSocket & {
    verified: boolean
}

type Api = {
    socket: VerifiableWebSocket | null
    openSocketRequest: Promise<boolean> | null
}

export const useApi = defineStore('api', {
    state: (): Api => {
        return {
            socket: null,
            openSocketRequest: null
        };
    },

    actions: {

        /**
         * Send a new API request
         * @param endpoint The relative endpoint
         * @param method The HTTP method
         * @param data Optional data to pass
         */
        fetchApi(endpoint: string, method: 'GET' | 'POST' | 'DELETE' | 'PATCH' | 'HEAD' = 'GET', data: any = null) {
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
        },

        /**
         * Get information about the server
         */
        async getInfo(): Promise<ApiResponse<Info>> {
            try {
                const raw = await this.fetchApi('/info');
                const data = await raw.json();
                return {
                    raw,
                    success: true,
                    data: {
                        platforms: data.platforms
                    }
                };
            } catch (error) {
                return {
                    success: false,
                    error
                };
            }
        },

        /**
         * Get information about a specific session
         * @param id The UUID of the session
         */
        async getSession(id: string): Promise<ApiResponse<SessionData>> {
            try {
                const raw = await this.fetchApi(`/sessions/${id}`);
                const data = await raw.json();

                return {
                    raw,
                    success: true,
                    data: {
                        request: {
                            links: data.request?.links,
                            platform: data.request?.platform,
                            platform_version: data.request?.platform_version,
                            game_version: data.request?.game_version
                        },
                        id: data.id,
                        links: data.links,
                        packages: data.packages,
                        overall_state: data.overall_state
                    }
                };
            } catch (error) {
                return {
                    success: false,
                    error
                };
            }
        },

        async newSessionRequest(request: Request): Promise<ApiResponse<{ id: string, links: Record<string, string> }>> {

            // Reformat the request so the links all have a unique ID
            const formattedRequest: Omit<Request, 'links'> & {
                links: Record<string, string>
            } = { ...request, links: {} };

            for (const link of request?.links?.split('\n') ?? []) {
                if (!link || !link.startsWith('http')) continue;

                let id = newUuid();
                while (formattedRequest.links[id]) id = newUuid();

                formattedRequest.links[id] = link;
            }

            // Send the request
            try {
                const raw = await this.fetchApi('/sessions', 'POST', formattedRequest);
                const data = await raw.json();

                return {
                    raw,
                    success: true,
                    data: {
                        id: data.id,
                        links: formattedRequest.links
                    }
                };
            } catch (error) {
                return {
                    success: false,
                    error,
                };
            }
        },

        /**
         * Handle messages from the server
         * @param message The main message or command that was sent
         * @param data Any additional data or null
         */
        handleMessage(message: string, data: any = null) {
            const store = useStore();

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
                    store.session.links[state.id] = state;
                    break;
                }

                // Whenever a stage is done, we'll update the rest of the session data
                case messages.PRELIMINARY_DONE:
                case messages.PACKAGE_DONE:
                case messages.PROCESS_DONE:
                case messages.GET_DOWNLOAD_DONE: {
                    store.updateSession(data as SessionData);
                    break;
                }

                case messages.DELETED: {
                    store.clearSession();
                    break;
                }

                default:
                    console.error('Unable to handle command:', message, data);
                    break;
            }
        },

        /**
         * Send a new message to the server
         * @param message The message or command to send
         * @param data The raw data to send or null if there is none
         */
        async sendMessage(message: string, data: any = null) {

            // If we don't have a socket, or it isn't yet verified, we'll open one first,
            // or if it's already in progress, await it
            if (!this.socket?.verified) {
                if (!await this.openSocket()) {
                    console.error('Unable to send message, openSocket return false: ', message, data);
                    return;
                }
            }

            // If the socket still isn't verified, there is nothing we can do
            if (!this.socket?.verified) {
                console.error('Unable to send message, socket not verified: ', message, data);
                return;
            }

            // Then we just serialize our message and send it
            let payload = `${message}`;
            if (data) payload += ` ${JSON.stringify(data)}`;
            this.socket.send(payload);
        },

        /**
         * Close the current websocket connection to free up resources
         */
        closeSocket() {
            if (!this.socket) return;
            try {
                this.socket.close();
                this.socket = null;
            } catch (e) {
                // Don't care, didn't ask
            }
        },

        /**
         * Open a new websocket for the session
         * This uses a global promise, so in case several consumers
         * call asynchronously, it should still return a single promise
         */
        openSocket(): Promise<boolean> {
            const store = useStore();

            // Return if there is an existing running promise
            if (this.openSocketRequest) return this.openSocketRequest;

            this.openSocketRequest = new Promise<boolean>(async resolve => {
                const sessionId = store.session.id;
                if (!sessionId) return resolve(false);
                if (this.socket && this.socket.verified) return resolve(true);

                const newSocket = new WebSocket(`${config.backend.ssl ? 'wss' : 'ws'}://${config.backend.endpoint}/api/sessions/${sessionId}/socket`) as VerifiableWebSocket;
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

                    // Handle the message
                    this.handleMessage(message, data);
                };

                newSocket.onerror = event => console.log('Socket error:', event);

                newSocket.onclose = () => {
                    this.socket = null;
                    if (resolve) resolve(false);
                };

                this.socket = newSocket;
            }).finally(() => {
                this.openSocketRequest = null;
            });

            return this.openSocketRequest;
        }

    }
});
