<template>
    <div v-loading="loading">
        <template v-if="summary">
            <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                    <div class="text-lg font-bold break-all">{{ summary.project.repo || summary.project.name }}</div>
                    <div class="text-xs break-all">{{ summary.project.cloneURL }}</div>
                    <div class="mt-1 flex items-center gap-2">
                        <el-tag size="small" effect="plain">{{ summary.project.branch }}</el-tag>
                        <span class="text-xs text-gray-400">{{ $t('github.defaultDeployBranch') }}</span>
                    </div>
                </div>
                <div class="flex shrink-0 gap-2">
                    <el-button icon="Setting" @click="settingsRef.acceptParams(summary.project)">
                        {{ $t('github.settings') }}
                    </el-button>
                    <el-button
                        v-permission
                        v-node-admin
                        type="primary"
                        icon="Refresh"
                        :disabled="summary.project.running"
                        @click="onDeployLatest"
                    >
                        {{ summary.project.running ? $t('github.running') : $t('github.deployLatest') }}
                    </el-button>
                </div>
            </div>

            <el-row :gutter="12" class="mt-4">
                <el-col :span="8">
                    <el-card shadow="never">
                        <div class="text-xs text-gray-400">{{ $t('github.deployedCommit') }}</div>
                        <div class="font-bold">{{ shortCommit(summary.commit) }}</div>
                        <div class="text-xs truncate">
                            {{ summary.commitAuthor ? summary.commitAuthor + ' · ' : '' }}{{ summary.commitMessage }}
                        </div>
                    </el-card>
                </el-col>
                <el-col :span="8">
                    <el-card shadow="never">
                        <div class="text-xs text-gray-400">{{ $t('github.lastDeployed') }}</div>
                        <div class="font-bold">{{ formatTime(summary.deployedAt || summary.project.lastRunAt) }}</div>
                        <div class="text-xs">{{ formatMillis(summary.duration) }}</div>
                    </el-card>
                </el-col>
                <el-col :span="8">
                    <el-card shadow="never">
                        <div class="text-xs text-gray-400">{{ $t('github.autoDeploy') }}</div>
                        <div class="flex items-center gap-2">
                            <el-tag size="small" :type="summary.project.webhookEnabled ? 'success' : 'info'">
                                {{ $t('github.tabWebhook') }}
                            </el-tag>
                            <span class="font-bold">
                                {{
                                    summary.project.webhookEnabled
                                        ? $t('github.enabledLabel')
                                        : $t('github.disabledLabel')
                                }}
                            </span>
                        </div>
                        <div class="flex items-center justify-between text-xs">
                            <span>{{ summary.project.branch }} · {{ $t('github.gitPush') }}</span>
                            <el-link type="primary" :underline="false" @click="emit('goto', 'webhook')">
                                {{ $t('github.manage') }} →
                            </el-link>
                        </div>
                    </el-card>
                </el-col>
            </el-row>

            <el-card shadow="never" class="mt-4">
                <div class="mb-3 flex items-center justify-between">
                    <div>
                        <div class="font-bold">{{ $t('github.commitHistory') }}</div>
                        <div class="text-xs text-gray-400">
                            {{ $t('github.remoteCommitsOn', [summary.project.branch]) }}
                        </div>
                    </div>
                    <el-button icon="Refresh" :loading="commitLoading" @click="loadCommits">
                        {{ $t('github.refresh') }}
                    </el-button>
                </div>
                <el-table :data="commits" v-loading="commitLoading" :empty-text="$t('github.notClonedYet')">
                    <el-table-column :label="$t('github.commit')" width="110">
                        <template #default="{ row }">
                            <span class="font-mono">{{ shortCommit(row.commit) }}</span>
                            <el-tag v-if="row.active" size="small" type="success" class="ml-1">
                                {{ $t('github.activeVersion') }}
                            </el-tag>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.message')" min-width="200" show-overflow-tooltip>
                        <template #default="{ row }">
                            {{ row.message }}
                            <span class="text-xs text-gray-400">· {{ row.author }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.commitOperate')" width="110" align="right">
                        <template #default="{ row }">
                            <el-button
                                v-permission
                                v-node-admin
                                link
                                type="primary"
                                :disabled="row.active || summary.project.running"
                                @click="onDeployCommit(row)"
                            >
                                {{ $t('github.deployCommit') }}
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>
                <el-pagination
                    v-if="total > pageSize"
                    class="mt-3 justify-end"
                    background
                    layout="prev, pager, next"
                    :total="total"
                    :page-size="pageSize"
                    v-model:current-page="page"
                    @current-change="loadCommits"
                />
            </el-card>
        </template>
        <RepoSettings ref="settingsRef" @saved="load" />
    </div>
</template>

<script lang="ts" setup>
// fork: Repository tab of the Git Manager: summary cards, commit history, deploy a commit.
import { onMounted, onUnmounted, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import { deployGithubProject, loadGithubSummary, searchGithubCommits } from '@/api/modules/github';
import RepoSettings from '@/views/github/components/git-manager/repo-settings.vue';
import { formatMillis, formatTime, shortCommit } from '@/views/github/helper';

const props = defineProps({
    projectId: {
        type: Number,
        required: true,
    },
});
const emit = defineEmits<{ (e: 'goto', tab: string): void }>();

const loading = ref(false);
const commitLoading = ref(false);
const summary = ref<Github.ProjectSummary>();
const commits = ref<Github.CommitInfo[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 10;
const settingsRef = ref();
let timer: ReturnType<typeof setInterval> | undefined;
let pollUntil = 0; // keep refreshing for a while after a deploy was started

const load = async (quiet = false) => {
    if (!quiet) loading.value = true;
    try {
        const res = await loadGithubSummary(props.projectId);
        const wasRunning = summary.value?.project.running;
        summary.value = res.data;
        if (wasRunning && !res.data.project.running) {
            loadCommits().catch(() => {});
        }
    } finally {
        loading.value = false;
    }
};

const loadCommits = async () => {
    if (!summary.value) return;
    commitLoading.value = true;
    try {
        const res = await searchGithubCommits({
            id: props.projectId,
            branch: summary.value.project.branch,
            page: page.value,
            pageSize: pageSize,
        });
        commits.value = res.data.items || [];
        total.value = res.data.total || 0;
    } catch {
        commits.value = [];
        total.value = 0;
    } finally {
        commitLoading.value = false;
    }
};

const onDeployLatest = () => {
    ElMessageBox.confirm(
        i18n.global.t('github.deployLatestHelper', [summary.value?.project.name]),
        i18n.global.t('github.deployLatest'),
        {
            confirmButtonText: i18n.global.t('commons.button.confirm'),
            cancelButtonText: i18n.global.t('commons.button.cancel'),
            type: 'warning',
        },
    ).then(async () => {
        await deployGithubProject({ id: props.projectId });
        MsgSuccess(i18n.global.t('github.deployStarted'));
        pollUntil = Date.now() + 15000;
        load(true);
    });
};

const onDeployCommit = (row: Github.CommitInfo) => {
    ElMessageBox.confirm(
        i18n.global.t('github.deployCommitConfirm', [shortCommit(row.commit)]),
        i18n.global.t('github.deployCommit'),
        {
            confirmButtonText: i18n.global.t('commons.button.confirm'),
            cancelButtonText: i18n.global.t('commons.button.cancel'),
            type: 'warning',
        },
    ).then(async () => {
        await deployGithubProject({ id: props.projectId, commit: row.commit });
        MsgSuccess(i18n.global.t('github.deployStarted'));
        pollUntil = Date.now() + 15000;
        load(true);
    });
};

onMounted(async () => {
    await load();
    loadCommits().catch(() => {});
    timer = setInterval(() => {
        if (summary.value?.project.running || Date.now() < pollUntil) {
            load(true).catch(() => {});
        }
    }, 3000);
});
onUnmounted(() => {
    if (timer) clearInterval(timer);
});
</script>
