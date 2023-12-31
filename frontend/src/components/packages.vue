<script setup lang="ts">
import { config, messages, useApi, useStore } from '@/helpers';

const store = useStore();
const api = useApi();
const session = store.session;

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
</script>

<template>
    <div v-if="session.packages" class="flex flex-row gap-3 justify-center items-center mt-3">
        <template v-for="[id, pkg] of Object.entries(session.packages)" :key="id">
            <div class="flex flex-row gap-3">
                <p class="text-red-400" v-if="pkg.status == 'error'">{{ pkg.message }}</p>
                <button class="btn primary"
                        @click="api.sendMessage(messages.GET_DOWNLOAD, id)"
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
</template>
