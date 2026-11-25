<template>
    <div v-loading="loading" class="flex flex-col gap-3">
        <el-card shadow="never">
            <div class="font-bold">{{ $t('github.webhookScriptCard') }}</div>
            <div class="mb-2 text-xs text-gray-400">{{ $t('github.webhookScriptCardHelper') }}</div>
            <div class="flex gap-2">
                <el-select v-model="selected" class="flex-1" :disabled="saving" @change="onSelect">
                    <el-option :value="0" :label="$t('github.noneDoNotDeploy')" />
                    <el-option v-for="item in scripts" :key="item.id" :value="item.id" :label="item.name" />
                </el-select>
                <el-button @click="emit('goto', 'script')">{{ $t('github.createScript') }}</el-button>
            </div>
        </el-card>

        <el-card shadow="never">
            <div class="flex items-start justify-between">
                <div>
                    <div class="font-bold">{{ $t('github.webhookUrlCard') }}</div>
                    <div class="mb-2 text-xs text-gray-400">{{ $t('github.webhookUrlHelper') }}</div>
                </div>
                <el-link
                    v-if="info?.enabled"
                    v-permission
                    v-node-admin
                    type="primary"
                    :underline="false"
                    @click="onRefreshToken"
                >
                    {{ $t('github.refreshToken') }}
                </el-link>
            </div>
            <div v-if="info?.enabled" class="flex gap-2">
                <el-input :model-value="url" readonly class="font-mono" />
                <CopyButton :content="url" :isIcon="false" />
            </div>
            <el-alert v-else type="info" :closable="false" :title="$t('github.webhookOffInfo')" />
            <div v-if="info?.enabled" class="mt-2 text-xs text-gray-400">{{ $t('github.webhookProviderHelper') }}</div>
        </el-card>

        <el-card shadow="never">
            <div class="flex items-start justify-between">
                <div>
                    <div class="font-bold">{{ $t('github.recentRequests') }}</div>
                    <div class="mb-2 text-xs text-gray-400">{{ $t('github.recentRequestsHelper') }}</div>
                </div>
                <div class="flex gap-2">
                    <el-button type="primary" size="small" @click="loadLast">{{ $t('github.refresh') }}</el-button>
                    <el-button size="small" v-permission v-node-admin @click="onClear">
                        {{ $t('github.clear') }}
                    </el-button>
                </div>
            </div>
            <LogFile
                v-if="last?.taskID"
                :key="last.taskID"
                :config="{ type: 'task', colorMode: 'task', taskID: last.taskID, tail: false }"
                :heightDiff="520"
            />
            <div v-else class="text-sm text-gray-400">
                {{ last?.message || $t('github.noWebhookRequests') }}
            </div>
        </el-card>
    </div>
</template>

<script lang="ts" setup>
// fork: Webhook tab: the script the webhook runs, its URL and the latest request.
import { computed, onMounted, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import LogFile from '@/components/log/file/index.vue';
import {
    cleanGithubLogs,
    configureGithubWebhook,
    listGithubScripts,
    loadGithubWebhook,
    refreshGithubWebhookToken,
    searchGithubLogs,
} from '@/api/modules/github';
import { webhookDisplayURL } from '@/views/github/helper';

const props = defineProps({
    projectId: {
        type: Number,
        required: true,
    },
});
const emit = defineEmits<{ (e: 'goto', tab: string): void }>();

const loading = ref(false);
const saving = ref(false);
const info = ref<Github.WebhookInfo>();
const scripts = ref<Github.ScriptInfo[]>([]);
const selected = ref<number | undefined>(0);
const last = ref<Github.LogInfo>();

const url = computed(() => webhookDisplayURL(info.value));

const load = async () => {
    loading.value = true;
    try {
        const [webhook, list] = await Promise.all([
            loadGithubWebhook(props.projectId),
            listGithubScripts(props.projectId),
        ]);
        info.value = webhook.data;
        scripts.value = list.data || [];
        if (!webhook.data.enabled) {
            selected.value = 0;
        } else {
            // A webhook that is on without a script shows the empty placeholder.
            selected.value = scripts.value.some((item) => item.id === webhook.data.scriptID)
                ? webhook.data.scriptID
                : undefined;
        }
    } finally {
        loading.value = false;
    }
};

const loadLast = async () => {
    const res = await searchGithubLogs({ projectID: props.projectId, trigger: 'webhook', page: 1, pageSize: 1 });
    last.value = (res.data.items || [])[0];
};

const onSelect = async (value: number) => {
    saving.value = true;
    try {
        await configureGithubWebhook({ id: props.projectId, enabled: value !== 0, scriptID: value });
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        await load();
    } catch {
        await load();
    } finally {
        saving.value = false;
    }
};

const onRefreshToken = () => {
    ElMessageBox.confirm(i18n.global.t('github.refreshTokenHelper'), i18n.global.t('github.refreshToken'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await refreshGithubWebhookToken(props.projectId);
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        await load();
    });
};

const onClear = () => {
    ElMessageBox.confirm(i18n.global.t('github.clearWebhookLogsHelper'), i18n.global.t('github.clear'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await cleanGithubLogs(props.projectId, 'webhook');
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        last.value = undefined;
        await loadLast();
    });
};

onMounted(() => {
    load().catch(() => {});
    loadLast().catch(() => {});
});
</script>
