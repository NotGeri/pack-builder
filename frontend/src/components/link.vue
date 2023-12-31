<script setup lang="ts">
import { computed, ref } from 'vue';
import type { LinkState } from '@/helpers';
import { errors, fixableErrors, messages, useApi } from '@/helpers';

const api = useApi();
const props = defineProps<LinkState>();

const advanced = ref<boolean>(false);
const warning = ref<boolean>(false);
const error = ref<boolean>(false);

if (props.preliminary?.status === 'error') {
    if (fixableErrors.includes(props.preliminary.error)) warning.value = true;
    else error.value = true;
}

const selectedVersion = computed(() => {
    return props.preliminary?.plugin_info?.versions[0] ?? null;
});
</script>

<template>
    <div
        :class="['bg-darkest p-5 rounded-xl flex flex-col gap-3 relative border-2', {'border-green-400': !warning && !error}, {'border-orange-400': warning}, {'border-red-400': error}]">
        <div class="absolute top-1 right-1 flex flex-col justify-end items-end text-right">
            <button @click="advanced = !advanced">
                ...
            </button>
            <div v-if="advanced" class="flex flex-col justify-end gap-2 text-xs mt-2">
                <button class="btn">Link Manually</button>
                <button class="btn">Skip</button>
            </div>
        </div>

        <a class="w-fit max-w-[90%]" target="_blank" :href="link">
            <h1 class="text-xl whitespace-nowrap overflow-hidden block">{{ link }}</h1>
        </a>

        <div v-if="preliminary" class="flex flex-col gap-5">
            <div v-if="preliminary.plugin_info">
                <h2 class="whitespace-nowrap overflow-hidden">{{ preliminary.plugin_info.name }}</h2>

                <p>
                    Link:
                    <a :href="preliminary.plugin_info.link" target="_blank">{{ preliminary.plugin_info.link }}</a>
                </p>
                <p v-if="preliminary.plugin_info.contributors">Contributors:
                    {{ preliminary.plugin_info.contributors }}</p>

                <!--
                Currently doing: figure out a way to handle problems, such as if there isn't a version
                also the ability to add new links if needed?
                1. [ ] redo the layout so they are rows, not columns
                -->

                <div v-if="preliminary.certain && selectedVersion">
                    <p v-if="selectedVersion.link">
                        Selected version:
                        <a :href="selectedVersion.link" target="_blank">{{ selectedVersion.link }}</a>
                    </p>
                    <p v-if="selectedVersion.game_versions">Supported versions:
                        {{ selectedVersion.game_versions.join(', ') }}
                    </p>
                </div>
            </div>

            <div v-if="preliminary.status === 'error'">
                <div v-if="preliminary.plugin_info && preliminary.error === errors.NO_SUITABLE_VERSION">
                    <p class="text-red-400">
                        No suitable version was found!
                    </p>
                </div>
                <p v-else class="text-xs text-red-400">
                    {{ preliminary.error }} {{ preliminary.message }}
                </p>
            </div>

            <div v-if="!preliminary?.certain && preliminary?.links" class="flex flex-col justify-center">

                <h3>Available Unverified Downloads</h3>

                <template v-for="[link, selected] of Object.entries(preliminary.links)" :key="link">
                    <div class="flex flex-row gap-1">
                        <input type="checkbox"
                               class="w-5"
                               v-model="preliminary.links[link]"
                               @change="api.sendMessage(messages.TOGGLE_LINK, { id: id, link, value: selected })">
                        <span class="text-xs">{{ link }}</span>
                    </div>
                </template>

                <p class="text-red-400 text-xs mt-1" v-if="Object.keys(preliminary.links).length == 0">
                    Please provide a download link!
                </p>
                <p class="text-red-400 text-xs mt-1" v-else-if="Object.values(preliminary.links).every(v => !v)">
                    Please ensure you provide at least one download link!
                </p>
            </div>
        </div>

        <div v-if="download">
            {{ download }}
        </div>

    </div>

</template>
