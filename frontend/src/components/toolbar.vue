<script setup lang="ts">
import { messages, useApi, useStore } from '@/helpers';

const store = useStore();
const api = useApi();
</script>

<template>
    <div v-if="store.session.id"
         class="flex flex-row gap-3 mt-10 justify-center items-center fixed bottom-10 left-1/2 transform -translate-x-1/2 bg-black p-4 rounded-xl">
        <button class="btn success"
                @click="api.sendMessage(messages.PRELIMINARY)">
            Preliminary
        </button>
        <button class="btn success"
                :disabled="!store.session?.overall_state?.preliminary"
                @click="api.sendMessage(messages.PROCESS)">
            Process
        </button>
        <button class="btn success"
                :disabled="!store.session?.overall_state?.download"
                @click="api.sendMessage(messages.PACKAGE)">
            Package
        </button>
        <button class="btn danger"
                @click="api.sendMessage(messages.DELETE)">
            Delete
        </button>
    </div>
</template>
