<template>
    <!-- fork: read-only view of the runtime that serves the website -->
    <div class="max-w-3xl" v-loading="loading">
        <el-descriptions v-if="runtime" border :column="1">
            <el-descriptions-item :label="$t('commons.table.name')">{{ runtime.name }}</el-descriptions-item>
            <el-descriptions-item :label="$t('commons.table.type')">{{ runtime.type }}</el-descriptions-item>
            <el-descriptions-item :label="$t('app.version')">{{ runtime.version }}</el-descriptions-item>
            <el-descriptions-item :label="$t('commons.table.status')">
                <RuntimeStatus :row="runtime" />
            </el-descriptions-item>
            <el-descriptions-item v-if="containerName" :label="$t('app.containerName')">
                {{ containerName }}
            </el-descriptions-item>
            <el-descriptions-item v-if="runtime.codeDir" :label="$t('runtime.codeDir')">
                <el-button type="primary" link @click="routerToFileWithPath(runtime.codeDir)">
                    {{ runtime.codeDir }}
                </el-button>
            </el-descriptions-item>
            <el-descriptions-item v-if="runScript" :label="$t('runtime.runScript')">
                <span class="break-all">{{ runScript }}</span>
            </el-descriptions-item>
            <el-descriptions-item v-if="ports.length > 0" :label="$t('commons.table.port')">
                <div v-for="port in ports" :key="port">{{ port }}</div>
            </el-descriptions-item>
            <el-descriptions-item v-if="proxy" :label="$t('website.proxyAddress')">
                {{ proxy }}
            </el-descriptions-item>
            <el-descriptions-item v-if="runtime.remark" :label="$t('website.remark')">
                {{ runtime.remark }}
            </el-descriptions-item>
        </el-descriptions>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Runtime } from '@/api/interface/runtime';
import { GetRuntime } from '@/api/modules/runtime';
import { getWebsite } from '@/api/modules/website';
import { routerToFileWithPath } from '@/utils/router';
import RuntimeStatus from '@/views/website/runtime/components/runtime-status.vue';

const props = defineProps({
    id: {
        type: Number,
        default: 0,
    },
});

const loading = ref(false);
const runtime = ref<Runtime.RuntimeDTO>();
const proxy = ref('');

// params comes back as an object here, and is absent for local PHP runtimes
const params = computed<Record<string, any>>(() => (runtime.value?.params || {}) as Record<string, any>);
const containerName = computed(() => params.value.CONTAINER_NAME || runtime.value?.container);
const runScript = computed(() => params.value.EXEC_SCRIPT);

// shown as the Docker mapping, an empty host IP means every interface
const ports = computed(() =>
    (runtime.value?.exposedPorts || []).map(
        (port) => `${port.hostIP || '0.0.0.0'}:${port.hostPort}->${port.containerPort}/${port.protocol || 'tcp'}`,
    ),
);

const search = async () => {
    loading.value = true;
    try {
        const website = await getWebsite(props.id);
        proxy.value = (website.data as any).proxy;
        const res = await GetRuntime(website.data.runtimeID);
        runtime.value = res.data;
    } catch (error) {
    } finally {
        loading.value = false;
    }
};

onMounted(() => {
    search();
});
</script>
