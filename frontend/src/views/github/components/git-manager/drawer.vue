<template>
    <DrawerPro v-model="visible" :header="$t('github.gitManager')" :resource="resource" size="large">
        <GitManager v-if="visible" :runtime-id="runtimeId" :website-id="websiteId" />
    </DrawerPro>
</template>

<script lang="ts" setup>
// fork: Git Manager in a drawer (runtime list).
import { ref } from 'vue';
import GitManager from '@/views/github/components/git-manager/index.vue';

const visible = ref(false);
const resource = ref('');
const runtimeId = ref(0);
const websiteId = ref(0);

const acceptParams = (params: { runtimeId?: number; websiteId?: number; name?: string }) => {
    runtimeId.value = params.runtimeId || 0;
    websiteId.value = params.websiteId || 0;
    resource.value = params.name || '';
    visible.value = true;
};

defineExpose({ acceptParams });
</script>
