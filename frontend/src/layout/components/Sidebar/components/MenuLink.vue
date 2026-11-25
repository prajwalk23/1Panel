<template>
    <a class="menu-link" :href="router.resolve(path).href" :aria-label="title" tabindex="-1" @click="handleClick"></a>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';

defineProps<{ path: string; title: string }>();

const router = useRouter();

// A real link gives the browser its "open in new tab" actions. Plain clicks stay with el-menu.
const handleClick = (event: MouseEvent) => {
    if (event.ctrlKey || event.metaKey || event.shiftKey || event.altKey) {
        event.stopPropagation();
        return;
    }
    event.preventDefault();
};
</script>

<style scoped lang="scss">
.menu-link {
    position: absolute;
    inset: 0;
    z-index: 1;
}
</style>
