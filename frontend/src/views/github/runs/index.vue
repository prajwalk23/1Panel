<template>
    <div>
        <LayoutContent v-loading="loading" :title="$t('github.logs')">
            <template #leftToolBar>
                <el-select v-model="projectID" class="!w-56" @change="search()">
                    <el-option :value="0" :label="$t('github.allProjects')" />
                    <el-option v-for="item in projects" :key="item.id" :value="item.id" :label="item.name" />
                </el-select>
                <el-select v-model="status" class="!w-40 ml-2" clearable @change="search()">
                    <el-option v-for="item in statusOptions" :key="item" :value="item" :label="statusLabel(item)" />
                </el-select>
                <el-button v-permission v-node-admin class="ml-2" @click="onClean">
                    {{ $t('github.cleanLogs') }}
                </el-button>
            </template>
            <template #rightToolBar>
                <TableRefresh @search="search()" />
            </template>
            <template #main>
                <ComplexTable
                    :cache-pagination="false"
                    :pagination-config="paginationConfig"
                    @search="search"
                    :data="data"
                    row-key="id"
                >
                    <el-table-column :label="$t('github.project')" prop="projectName" min-width="120" />
                    <el-table-column :label="$t('github.trigger')" min-width="90">
                        <template #default="{ row }">
                            {{ row.trigger === 'webhook' ? $t('github.triggerWebhook') : $t('github.triggerManual') }}
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.commit')" min-width="100">
                        <template #default="{ row }">
                            <span class="font-mono">{{ shortCommit(row.commit) }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.pusher')" prop="pusher" min-width="100">
                        <template #default="{ row }">{{ row.pusher || '-' }}</template>
                    </el-table-column>
                    <el-table-column :label="$t('github.status')" min-width="100">
                        <template #default="{ row }">
                            <el-tooltip v-if="row.message" :content="row.message" placement="top">
                                <el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
                            </el-tooltip>
                            <el-tag v-else :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.startedAt')" min-width="160">
                        <template #default="{ row }">{{ formatTime(row.startedAt || row.createdAt) }}</template>
                    </el-table-column>
                    <el-table-column :label="$t('github.duration')" min-width="90">
                        <template #default="{ row }">{{ formatDuration(row.startedAt, row.finishedAt) }}</template>
                    </el-table-column>
                    <fu-table-operations width="120px" :buttons="buttons" :label="$t('commons.table.operate')" fix />
                </ComplexTable>
            </template>
        </LayoutContent>

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
// fork: deploy runs of all projects.
import { onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import LogFile from '@/components/log/file/index.vue';
import { Github } from '@/api/interface/github';
import { cleanGithubLogs, searchGithubLogs, searchGithubProjects } from '@/api/modules/github';
import { formatDuration, formatTime, shortCommit, statusLabel, statusType } from '@/views/github/helper';

const route = useRoute();
const loading = ref(false);
const data = ref<Github.LogInfo[]>([]);
const projects = ref<Github.ProjectInfo[]>([]);
const projectID = ref(Number(route.query.projectID) || 0);
const status = ref('');
const statusOptions = ['waiting', 'running', 'success', 'failed', 'canceled', 'skipped'];
const paginationConfig = reactive({ currentPage: 1, pageSize: 20, total: 0 });
const logVisible = ref(false);
const logTaskID = ref('');
const logTitle = ref('');

const search = async () => {
    loading.value = true;
    try {
        const res = await searchGithubLogs({
            projectID: projectID.value || undefined,
            status: status.value || undefined,
            page: paginationConfig.currentPage,
            pageSize: paginationConfig.pageSize,
        });
        data.value = res.data.items || [];
        paginationConfig.total = res.data.total;
    } finally {
        loading.value = false;
    }
};

const loadProjects = async () => {
    const res = await searchGithubProjects({ info: '', page: 1, pageSize: 1000 });
    projects.value = res.data.items || [];
};

const onClean = () => {
    const target = projects.value.find((item) => item.id === projectID.value);
    const name = target ? target.name : i18n.global.t('github.allProjects');
    ElMessageBox.confirm(i18n.global.t('github.cleanLogsHelper', [name]), i18n.global.t('github.cleanLogs'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await cleanGithubLogs(projectID.value);
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        search();
    });
};

const buttons = [
    {
        label: i18n.global.t('github.viewLog'),
        disabled: (row: Github.LogInfo) => !row.taskID,
        click: (row: Github.LogInfo) => {
            logTaskID.value = row.taskID;
            logTitle.value = row.projectName;
            logVisible.value = true;
        },
    },
];

onMounted(() => {
    loadProjects().catch(() => {});
    search();
});
</script>
