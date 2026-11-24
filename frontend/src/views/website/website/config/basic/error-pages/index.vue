<template>
    <el-row :gutter="20" v-loading="loading">
        <el-col :xs="24" :sm="18" :md="14" :lg="14" :xl="14">
            <el-form @submit.prevent label-position="top">
                <el-form-item>
                    <el-checkbox v-model="useGlobal" :disabled="loading" :before-change="changeUseGlobal">
                        {{ $t('github.siteUseGlobalErrorPages') }}
                    </el-checkbox>
                    <div>
                        <el-text type="info" size="small">{{ $t('github.siteUseGlobalErrorPagesHelper') }}</el-text>
                    </div>
                </el-form-item>
            </el-form>
        </el-col>
    </el-row>
</template>

<script lang="ts" setup>
// fork: per-website switch for the shared custom error pages (default: follow the global setting)
import { getWebsite, setSiteErrorPages } from '@/api/modules/website';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { onMounted, ref } from 'vue';

const props = defineProps({
    id: {
        type: Number,
        default: 0,
    },
});

const loading = ref(false);
const useGlobal = ref(true);

const load = async () => {
    loading.value = true;
    try {
        const res = await getWebsite(props.id);
        useGlobal.value = !res.data.errorPagesOff;
    } finally {
        loading.value = false;
    }
};

// the checkbox only changes when this resolves true
const changeUseGlobal = async () => {
    loading.value = true;
    try {
        await setSiteErrorPages({ websiteID: props.id, useGlobal: !useGlobal.value });
        MsgSuccess(i18n.global.t('commons.msg.updateSuccess'));
        return true;
    } catch {
        return false;
    } finally {
        loading.value = false;
    }
};

onMounted(load);
</script>
