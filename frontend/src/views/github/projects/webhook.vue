<template>
    <DialogPro v-model="visible" :title="$t('github.webhookTitle')" size="large" @close="onClose">
        <div v-loading="loading">
            <el-alert v-if="info && !info.enabled" type="info" :closable="false" class="mb-3">
                <template #title>{{ $t('github.webhookOffInfo') }}</template>
            </el-alert>
            <template v-else>
                <el-alert v-if="warning" type="warning" :closable="false" class="mb-3" :title="warning" />
                <el-alert v-if="info && !info.https && info.url" type="warning" :closable="false" class="mb-3">
                    <template #title>{{ $t('github.notHTTPS') }}</template>
                </el-alert>

                <p class="mb-2">{{ $t('github.githubSteps') }}</p>
                <el-descriptions :column="1" border>
                    <el-descriptions-item :label="$t('github.payloadURL')">
                        <div class="flex items-center gap-1 break-all">
                            <span>{{ info?.url || '-' }}</span>
                            <CopyButton v-if="info?.url" :content="info.url" />
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('github.contentType')">application/json</el-descriptions-item>
                    <el-descriptions-item :label="$t('github.secret')">
                        <div v-if="secret" class="flex flex-col gap-1">
                            <div class="flex items-center gap-1 break-all font-mono">
                                <span>{{ secret }}</span>
                                <CopyButton :content="secret" />
                            </div>
                            <span class="text-xs text-orange-500">{{ $t('github.secretOnce') }}</span>
                        </div>
                        <div v-else class="flex items-center gap-2">
                            <span class="text-gray-400">{{ $t('github.secretHidden') }}</span>
                            <el-button v-permission v-node-admin link type="primary" @click="onRegenerate">
                                {{ $t('github.regenerateSecret') }}
                            </el-button>
                        </div>
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('github.events')">
                        {{ $t('github.eventsValue') }}
                    </el-descriptions-item>
                    <el-descriptions-item :label="$t('github.webhookScript')">
                        {{ info?.scriptName || '-' }}
                    </el-descriptions-item>
                </el-descriptions>

                <template v-if="info?.proxyPass">
                    <p class="mt-4 mb-2">{{ $t('github.proxyHelper') }}</p>
                    <div class="flex items-start gap-1">
                        <el-input :model-value="proxySnippet" type="textarea" :rows="6" readonly class="font-mono" />
                        <CopyButton :content="proxySnippet" />
                    </div>
                </template>

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
// fork: shows what to enter on GitHub for a project's webhook and deploy key.
import { computed, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { Github } from '@/api/interface/github';
import { loadGithubWebhook, regenerateGithubSecret, searchGithubProjects } from '@/api/modules/github';

const emit = defineEmits<{ (e: 'search'): void }>();

const visible = ref(false);
const loading = ref(false);
const projectID = ref(0);
const info = ref<Github.WebhookInfo>();
const secret = ref('');
const publicKey = ref('');

const warning = computed(() => {
    if (!info.value) {
        return '';
    }
    if (!info.value.url) {
        return i18n.global.t('github.noDomain');
    }
    if (!info.value.listener) {
        return i18n.global.t('github.listenerDown');
    }
    return '';
});

const proxySnippet = computed(() => {
    const pass = info.value?.proxyPass || '';
    return [
        'location ^~ /github-hook/ {',
        `    proxy_pass ${pass};`,
        '    proxy_set_header Host $host;',
        '    proxy_set_header X-Real-IP $remote_addr;',
        '    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;',
        '}',
    ].join('\n');
});

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

const acceptParams = (id: number, created?: Github.ProjectSecret) => {
    projectID.value = id;
    secret.value = created?.secret || '';
    publicKey.value = created?.publicKey || '';
    info.value = undefined;
    visible.value = true;
    load().catch(() => {});
};

const onRegenerate = () => {
    ElMessageBox.confirm(i18n.global.t('github.regenerateSecretHelper'), i18n.global.t('github.regenerateSecret'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        const res = await regenerateGithubSecret(projectID.value);
        secret.value = res.data.secret;
        emit('search');
    });
};

const onClose = () => {
    secret.value = '';
};

defineExpose({ acceptParams });
</script>
