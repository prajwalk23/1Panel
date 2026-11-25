<template>
    <el-divider content-position="left">
        <el-text type="info" size="small">{{ $t('website.createWebsiteTitle') }}</el-text>
    </el-divider>
    <el-form-item>
        <el-checkbox v-model="runtime.createWebsite" :label="$t('website.createWebsiteWith')" size="large" />
        <span class="input-help">{{ $t('website.createWebsiteHelper') }}</span>
    </el-form-item>
    <div v-if="runtime.createWebsite">
        <GroupSelect v-model="runtime.webSiteGroupId" prop="webSiteGroupId" groupType="website" />
        <DomainCreate v-model:form="runtime" :hide-ssl="true" />
        <el-form-item :label="$t('website.alias')" prop="alias" :rules="Rules.alias" class="mt-2">
            <el-input v-model.trim="runtime.alias" :placeholder="$t('website.aliasHelper')" @input="onAliasInput" />
        </el-form-item>
        <el-form-item :label="$t('website.createWebsiteProxyPort')">
            <el-select v-if="ports.length > 0" v-model="runtime.websitePort" class="p-w-200">
                <el-option v-for="port in ports" :key="port" :label="port" :value="port" />
            </el-select>
            <el-alert v-else type="warning" :closable="false" show-icon :title="$t('website.createWebsiteNoPort')" />
        </el-form-item>
    </div>
</template>

<script lang="ts" setup>
import GroupSelect from '@/views/website/website/components/group/index.vue';
import DomainCreate from '@/views/website/website/domain-create/index.vue';
import { Rules } from '@/global/form-rules';
import { useVModel } from '@vueuse/core';
import { computed, ref, watch, type PropType } from 'vue';

const props = defineProps({
    modelValue: {
        type: Object as PropType<any>,
        required: true,
    },
});
const emit = defineEmits(['update:modelValue']);
const runtime = useVModel(props, 'modelValue', emit);

// website proxies to one of the host ports exposed by the runtime
const ports = computed<number[]>(() =>
    ((runtime.value.exposedPorts || []) as any[])
        .map((item) => Number(item.hostPort))
        .filter((port, index, all) => port > 0 && all.indexOf(port) === index),
);

watch(
    ports,
    (list) => {
        if (!list.includes(runtime.value.websitePort)) {
            runtime.value.websitePort = list.length > 0 ? list[0] : 0;
        }
    },
    { immediate: true },
);

// alias follows the first domain until the user types one
const aliasManual = ref(false);
const onAliasInput = (value: string) => {
    aliasManual.value = value.trim() !== '';
};

watch(
    () => runtime.value.domains?.[0]?.domain,
    (value) => {
        if (aliasManual.value) {
            return;
        }
        runtime.value.alias = (value || '').split(':')[0].trim();
    },
);

watch(
    () => runtime.value.createWebsite,
    (enabled) => {
        if (!enabled) {
            aliasManual.value = false;
        }
    },
);
</script>
