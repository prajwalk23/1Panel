<template>
    <div v-loading="loading">
        <el-card>
            <template #header>
                <span>{{ $t('github.settings') }}</span>
            </template>
            <el-alert v-if="info && !info.gitInstalled" type="warning" :closable="false" class="mb-3">
                <template #title>{{ $t('github.gitMissing') }}</template>
            </el-alert>
            <el-form ref="formRef" label-position="top" :model="form" :rules="rules" class="max-w-3xl">
                <el-form-item :label="$t('github.webhookBase')">
                    <div class="flex items-center gap-2">
                        <span class="break-all">{{ info?.webhookBase || '-' }}</span>
                        <el-tag :type="info?.bindDomain ? 'success' : 'info'">
                            {{ info?.bindDomain ? $t('github.webhookBoundDomain') : $t('github.webhookNoDomain') }}
                        </el-tag>
                    </div>
                    <span class="input-help">{{ $t('github.webhookBaseHelper') }}</span>
                </el-form-item>

                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-form-item :label="$t('github.defaultShell')" prop="defaultShell">
                            <el-select v-model="form.defaultShell" class="w-full">
                                <el-option value="bash" label="bash" />
                                <el-option value="sh" label="sh" />
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item :label="$t('github.logRetention')" prop="logRetention">
                            <el-input-number v-model="form.logRetention" :min="1" :max="10000" />
                        </el-form-item>
                    </el-col>
                </el-row>
                <el-form-item>
                    <el-button v-permission v-node-admin type="primary" :disabled="loading" @click="onSave">
                        {{ $t('commons.button.save') }}
                    </el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </div>
</template>

<script lang="ts" setup>
// fork: GitHub section settings (webhook address, deploy shell, run history).
import { onMounted, reactive, ref } from 'vue';
import { ElForm } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import { loadGithubSetting, updateGithubSetting } from '@/api/modules/github';

type FormInstance = InstanceType<typeof ElForm>;

const loading = ref(false);
const formRef = ref<FormInstance>();
const info = ref<Github.SettingInfo>();
const form = reactive<Github.Setting>({
    defaultShell: 'bash',
    logRetention: 50,
    sharedKeyID: 0,
});

const rules = reactive({
    defaultShell: [Rules.requiredSelect],
});

const load = async () => {
    loading.value = true;
    try {
        const settingRes = await loadGithubSetting();
        info.value = settingRes.data;
        Object.assign(form, {
            defaultShell: settingRes.data.defaultShell,
            logRetention: settingRes.data.logRetention,
            sharedKeyID: settingRes.data.sharedKeyID,
        });
    } finally {
        loading.value = false;
    }
};

const onSave = async () => {
    if (!formRef.value) return;
    await formRef.value.validate(async (valid) => {
        if (!valid) return;
        loading.value = true;
        try {
            await updateGithubSetting({ ...form });
            MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        } finally {
            loading.value = false;
        }
        load().catch(() => {});
    });
};

onMounted(() => {
    load().catch(() => {});
});
</script>
