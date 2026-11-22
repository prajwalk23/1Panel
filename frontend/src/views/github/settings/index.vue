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
                <el-form-item :label="$t('github.listener')">
                    <el-tag v-if="info?.listenerRunning" type="success">
                        {{ $t('github.listenerRunning', [info.listenerAddr]) }}
                    </el-tag>
                    <template v-else>
                        <el-tag type="danger">{{ $t('github.listenerStopped') }}</el-tag>
                        <span v-if="info?.listenerError" class="ml-2 text-xs text-red-500">
                            {{ info.listenerError }}
                        </span>
                    </template>
                </el-form-item>
                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-form-item :label="$t('github.webhookBind')" prop="webhookBind">
                            <el-input v-model.trim="form.webhookBind" />
                            <span class="input-help">{{ $t('github.webhookBindHelper') }}</span>
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item :label="$t('github.webhookPort')" prop="webhookPort">
                            <el-input-number v-model="form.webhookPort" :min="1" :max="65535" />
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-form-item :label="$t('github.webhookDomain')" prop="webhookDomain">
                    <el-input v-model.trim="form.webhookDomain" placeholder="hooks.example.com" />
                    <span class="input-help">{{ $t('github.webhookDomainHelper') }}</span>
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
// fork: GitHub section settings (listener, webhook domain, deploy shell).
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
    webhookPort: 19090,
    webhookBind: '127.0.0.1',
    defaultShell: 'bash',
    logRetention: 50,
    webhookDomain: '',
    sharedKeyID: 0,
});

const checkBind = (rule: any, value: string, callback: any) => {
    const ipv4 = /^(\d{1,3}\.){3}\d{1,3}$/;
    if (!value || (!ipv4.test(value) && !value.includes(':'))) {
        callback(new Error(i18n.global.t('github.webhookBindHelper')));
        return;
    }
    callback();
};
const checkWebhookDomain = (rule: any, value: string, callback: any) => {
    if (value && !/^(https?:\/\/)?[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?(:\d{1,5})?\/?$/.test(value)) {
        callback(new Error(i18n.global.t('github.webhookDomainHelper')));
        return;
    }
    callback();
};
const rules = reactive({
    webhookBind: [{ validator: checkBind, trigger: 'blur', required: true }],
    webhookPort: [Rules.requiredInput],
    defaultShell: [Rules.requiredSelect],
    webhookDomain: [{ validator: checkWebhookDomain, trigger: 'blur' }],
});

const load = async () => {
    loading.value = true;
    try {
        const settingRes = await loadGithubSetting();
        info.value = settingRes.data;
        Object.assign(form, {
            webhookPort: settingRes.data.webhookPort,
            webhookBind: settingRes.data.webhookBind,
            defaultShell: settingRes.data.defaultShell,
            logRetention: settingRes.data.logRetention,
            webhookDomain: settingRes.data.webhookDomain,
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
