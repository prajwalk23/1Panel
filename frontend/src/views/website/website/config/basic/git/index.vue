<template>
    <GitManager :website-id="id" />
</template>

<script lang="ts" setup>
// fork: Git tab of the website settings. The Git Manager of the repository the site was created from.
import GitManager from '@/views/github/components/git-manager/index.vue';

defineProps({
    id: {
        type: Number,
        required: true,
    },
});
</script>
