<template>
    <DrawerPro v-model="drawerVisible" :header="$t('memoryLimit.title')" @close="handleClose" size="small">
        <el-form ref="formRef" label-position="top" :rules="rules" :model="form" @submit.prevent v-loading="loading">
            <el-form-item :label="$t('memoryLimit.usage')">
                <div class="usage-list">
                    <div v-for="item in usageItems" :key="item.key" class="usage-item">
                        <div class="usage-head">
                            <span>{{ item.label }}</span>
                            <span v-if="item.running" :class="{ 'is-over': item.over }">
                                {{ item.rss.toFixed(0) }} MiB / {{ info.limitMiB }} MiB
                                <template v-if="item.heap !== null">
                                    · {{ $t('memoryLimit.heap') }} {{ item.heap.toFixed(0) }} MiB
                                </template>
                            </span>
                            <span v-else class="input-help">{{ $t('memoryLimit.notRunning') }}</span>
                        </div>
                        <el-progress
                            v-if="item.running"
                            :percentage="item.percent"
                            :status="item.over ? 'exception' : ''"
                            :show-text="false"
                            :stroke-width="8"
                        />
                    </div>
                </div>
            </el-form-item>

            <el-form-item :label="$t('memoryLimit.limit')" prop="limitMiB">
                <el-input v-model.number="form.limitMiB" />
                <span class="input-help">
                    {{ $t('memoryLimit.range', [info.minMiB, info.maxMiB, info.defaultMiB]) }}
                    {{ $t('memoryLimit.appliedHelper') }}
                </span>
                <span class="input-help">{{ $t('memoryLimit.help') }}</span>
                <el-alert v-if="info.envOverride" type="warning" :closable="false" class="mt-2">
                    {{ $t('memoryLimit.envOverride') }}
                </el-alert>
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="drawerVisible = false">{{ $t('commons.button.cancel') }}</el-button>
            <el-button :disabled="loading" type="primary" @click="onSave(formRef)">
                {{ $t('commons.button.confirm') }}
            </el-button>
        </template>
    </DrawerPro>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from 'vue';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { FormInstance } from 'element-plus';
import { Rules, checkNumberRange } from '@/global/form-rules';
import { getPanelMemoryLimit, updatePanelMemoryLimit, MemoryLimit } from '@/api/modules/memory-limit';

const emit = defineEmits<{ (e: 'search'): void }>();

const drawerVisible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();

const info = reactive<MemoryLimit.Info>({
    limitMiB: 512,
    defaultMiB: 512,
    minMiB: 128,
    maxMiB: 262144,
    envOverride: false,
    core: { running: false, rssMiB: 0, heapMiB: 0 },
    agent: { running: false, rssMiB: 0, heapMiB: 0 },
});
const form = reactive({ limitMiB: 512 });

const rules = computed(() => ({
    limitMiB: [Rules.requiredInput, Rules.integerNumber, checkNumberRange(info.minMiB, info.maxMiB)],
}));

const usageItems = computed(() =>
    [
        { key: 'core', label: i18n.global.t('memoryLimit.core'), data: info.core, heap: info.core.heapMiB },
        { key: 'agent', label: i18n.global.t('memoryLimit.agent'), data: info.agent, heap: null as number | null },
    ].map((item) => {
        const percent = info.limitMiB > 0 ? Math.min(100, Math.round((item.data.rssMiB / info.limitMiB) * 100)) : 0;
        return {
            key: item.key,
            label: item.label,
            running: item.data.running,
            rss: item.data.rssMiB,
            heap: item.heap,
            percent,
            over: item.data.rssMiB > info.limitMiB,
        };
    }),
);

const load = async () => {
    loading.value = true;
    try {
        const res = await getPanelMemoryLimit();
        Object.assign(info, res.data);
        form.limitMiB = res.data.limitMiB;
    } finally {
        loading.value = false;
    }
};

const acceptParams = async () => {
    drawerVisible.value = true;
    await load();
};

const onSave = (formEl: FormInstance | undefined) => {
    if (!formEl) return;
    formEl.validate(async (valid) => {
        if (!valid) return;
        loading.value = true;
        try {
            await updatePanelMemoryLimit(form.limitMiB);
            MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
            drawerVisible.value = false;
            emit('search');
        } finally {
            loading.value = false;
        }
    });
};

const handleClose = () => {
    drawerVisible.value = false;
};

defineExpose({ acceptParams });
</script>

<style scoped lang="scss">
.usage-list {
    width: 100%;
}
.usage-item {
    margin-bottom: 10px;
}
.usage-head {
    display: flex;
    justify-content: space-between;
    margin-bottom: 4px;
}
.is-over {
    color: var(--el-color-danger);
}
</style>
