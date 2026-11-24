<template>
    <DialogPro v-model="visible" :title="$t('github.webhookTitle')" size="large">
        <div v-loading="loading">
            <el-alert v-if="info && !info.enabled" type="info" :closable="false" class="mb-3">
                <template #title>{{ $t('github.webhookOffInfo') }}</template>
            </el-alert>
            <template v-else>
                <p class="mb-2">{{ $t('github.webhookUrlHelper') }}</p>
                <el-descriptions :column="1" border>
                    <el-descriptions-item :label="$t('github.payloadURL')">
                        <div class="flex items-center gap-1 break-all">
                            <span>{{ url || '-' }}</span>
                            <CopyButton v-if="url" :content="url" />
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('github.contentType')">application/json</el-descriptions-item>
                    <el-descriptions-item :label="$t('github.events')">
                        {{ $t('github.eventsValue') }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('github.webhookScript')">
                        {{ info?.scriptName || '-' }}
                    </el-descriptions-item>
                </el-descriptions>
                <p class="input-help mt-2">{{ $t('github.webhookProviderHelper') }}</p>
                <el-button v-permission v-node-admin class="mt-3" @click="onRefresh">
                    {{ $t('github.refreshToken') }}
                </el-button>

                <template v-if="publicKey">
                    <p class="mt-4 mb-2">{{ $t('github.deployKeySteps') }}</p>
                    <div class="flex items-start gap-1">
                        <el-input :model-value="publicKey" type="textarea" :rows="3" readonly class="font-mono" />
                        <CopyButton :content="publicKey" />
                    </div>
                </template>
            </template>
        </div>
        <template #footer>
            <el-button @click="visible = false">{{ $t('commons.button.close') }}</el-button>
        </template>
    </DialogPro>
</template>

<script lang="ts" setup>
// fork: shows the webhook URL of a project. The host is the bound domain when there is one,
// otherwise the address the panel is opened with.
import { computed, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { Github } from '@/api/interface/github';
import { loadGithubWebhook, refreshGithubWebhookToken, searchGithubProjects } from '@/api/modules/github';
import { webhookDisplayURL } from '@/views/github/helper';

const emit = defineEmits<{ (e: 'search'): void }>();

const visible = ref(false);
const loading = ref(false);
const projectID = ref(0);
const info = ref<Github.WebhookInfo>();
const publicKey = ref('');

const url = computed(() => webhookDisplayURL(info.value));

const load = async () => {
    loading.value = true;
    try {
        const res = await loadGithubWebhook(projectID.value);
        info.value = res.data;
        if (!publicKey.value) {
            const list = await searchGithubProjects({ info: '', page: 1, pageSize: 1000 });
            const project = (list.data.items || []).find((item) => item.id === projectID.value);
            publicKey.value = project?.publicKey || '';
        }
    } finally {
        loading.value = false;
    }
};

const acceptParams = (id: number) => {
    projectID.value = id;
    publicKey.value = '';
    info.value = undefined;
    visible.value = true;
    load().catch(() => {});
};

const onRefresh = () => {
    ElMessageBox.confirm(i18n.global.t('github.refreshTokenHelper'), i18n.global.t('github.refreshToken'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await refreshGithubWebhookToken(projectID.value);
        await load();
        emit('search');
    });
};

defineExpose({ acceptParams });
</script>
