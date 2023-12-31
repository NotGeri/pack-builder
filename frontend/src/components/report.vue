<script setup lang="ts">
import { ref } from 'vue';
import { errors, useStore } from '@/helpers';

const store = useStore();
const session = store.session;
const report = ref<Report | undefined>();

type LinkReport = {};
type Report = {
    links: Record<string, LinkReport>
    toClient?: string
};

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
</script>

<template>
    <div v-if="report" class="mt-10 flex flex-col justify-center items-center gap-3">
        <h1>Generated Report</h1>
        <textarea :value="report.toClient" :rows="report.toClient?.split('\n').length ?? 10"></textarea>
    </div>
</template>
