<template>
    <div>
        <div class="mb-3 flex items-start justify-between gap-3">
            <div>
                <div class="text-base font-bold">{{ $t('github.deploymentHistory') }}</div>
                <div class="text-xs text-gray-400">{{ $t('github.deploymentHistoryHelper') }}</div>
            </div>
            <div class="flex gap-2">
                <el-select
                    v-model="trigger"
                    clearable
                    class="!w-40"
                    :placeholder="$t('github.allTriggers')"
                    @change="search"
                >
                    <el-option v-for="item in triggers" :key="item" :value="item" :label="triggerLabel(item)" />
                </el-select>
                <el-select
                    v-model="status"
                    clearable
                    class="!w-36"
                    :placeholder="$t('github.allStatus')"
                    @change="search"
                >
                    <el-option v-for="item in statuses" :key="item" :value="item" :label="statusLabel(item)" />
                </el-select>
            </div>
        </div>
        <el-table :data="data" v-loading="loading">
            <el-table-column :label="$t('github.commit')" min-width="130">
                <template #default="{ row }">
                    <span class="font-mono">{{ shortCommit(row.commit) }}</span>
                    <div class="text-xs text-gray-400 truncate">
                        {{ row.commitMessage }}{{ row.commitAuthor ? ' · ' + row.commitAuthor : '' }}
                    </div>
                </template>
            </el-table-column>
            <el-table-column :label="$t('github.deployment')" min-width="150">
                <template #default="{ row }">
                    <span>{{ triggerLabel(row.trigger) }}</span>
                    <el-tag v-if="row.active" size="small" type="success" class="ml-1">
                        {{ $t('github.activeVersion') }}
                    </el-tag>
                    <div>
                        <el-tooltip v-if="row.message" :content="row.message" placement="top">
                            <el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
                        </el-tooltip>
                        <el-tag v-else size="small" :type="statusType(row.status)">
                            {{ statusLabel(row.status) }}
                        </el-tag>
                    </div>
                </template>
            </el-table-column>
            <el-table-column :label="$t('github.time')" min-width="150">
                <template #default="{ row }">
                    <div>{{ formatTime(row.startedAt || row.createdAt) }}</div>
                    <div class="text-xs text-gray-400">{{ formatMillis(row.duration) }}</div>
                </template>
            </el-table-column>
            <el-table-column :label="$t('commons.table.operate')" width="140" align="right">
                <template #default="{ row }">
                    <el-button link type="primary" :disabled="!row.taskID" @click="onLog(row)">
                        {{ $t('github.viewLog') }}
                    </el-button>
                    <el-button
                        v-if="row.rollbackable"
                        v-permission
                        v-node-admin
                        link
                        type="primary"
                        @click="onRollback(row)"
                    >
                        {{ $t('github.rollback') }}
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
            @current-change="search"
        />
        <DrawerPro v-model="logVisible" :header="$t('github.viewLog')" :resource="logTitle" size="large">
            <LogFile
                v-if="logVisible && logTaskID"
                :config="{ type: 'task', colorMode: 'task', taskID: logTaskID, tail: true }"
                :heightDiff="200"
            />
        </DrawerPro>
    </div>
</template>

<script lang="ts" setup>
// fork: Deployments tab: every pull and rollback of the project, the active version, rollback.
import { onMounted, onUnmounted, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import { deployGithubProject, searchGithubLogs } from '@/api/modules/github';
import LogFile from '@/components/log/file/index.vue';
import { formatMillis, formatTime, shortCommit, statusLabel, statusType, triggerLabel } from '@/views/github/helper';

const props = defineProps({
    projectId: {
        type: Number,
        required: true,
    },
});

const triggers = ['manual', 'webhook', 'rollback', 'create', 'boot'];
const statuses = ['success', 'failed', 'running', 'waiting', 'canceled', 'skipped'];

const loading = ref(false);
const data = ref<Github.LogInfo[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 10;
const trigger = ref('');
const status = ref('');
const logVisible = ref(false);
const logTaskID = ref('');
const logTitle = ref('');
let timer: ReturnType<typeof setInterval> | undefined;

const search = async (quiet = false) => {
    if (quiet !== true) loading.value = true;
    try {
        const res = await searchGithubLogs({
            projectID: props.projectId,
            trigger: trigger.value || undefined,
            status: status.value || undefined,
            page: page.value,
            pageSize: pageSize,
        });
        data.value = res.data.items || [];
        total.value = res.data.total || 0;
    } finally {
        loading.value = false;
    }
};

const onLog = (row: Github.LogInfo) => {
    logTaskID.value = row.taskID;
    logTitle.value = shortCommit(row.commit);
    logVisible.value = true;
};

const onRollback = (row: Github.LogInfo) => {
    ElMessageBox.confirm(
        i18n.global.t('github.rollbackConfirm', [shortCommit(row.commit)]),
        i18n.global.t('github.rollback'),
        {
            confirmButtonText: i18n.global.t('commons.button.confirm'),
            cancelButtonText: i18n.global.t('commons.button.cancel'),
            type: 'warning',
        },
    ).then(async () => {
        await deployGithubProject({ id: props.projectId, commit: row.commit, rollback: true });
        MsgSuccess(i18n.global.t('github.deployStarted'));
        search();
    });
};

onMounted(() => {
    search();
    timer = setInterval(() => {
        if (data.value.some((item) => item.status === 'running' || item.status === 'waiting')) {
            search(true).catch(() => {});
        }
    }, 4000);
});
onUnmounted(() => {
    if (timer) clearInterval(timer);
});
</script>
