<template>
    <div>
        <RouterButton :buttons="buttons" />
        <LayoutContent>
            <RouterViewCache />
        </LayoutContent>
    </div>
</template>

<script lang="ts" setup>
// fork: GitHub section wrapper with the sub-page tab bar.
import i18n from '@/lang';

const buttons = [
    { label: i18n.global.t('github.projects'), path: '/github/projects' },
    { label: i18n.global.t('github.logs'), path: '/github/logs' },
    { label: i18n.global.t('github.keys'), path: '/github/keys' },
    { label: i18n.global.t('github.settings'), path: '/github/settings' },
];
</script>
