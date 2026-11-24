<template>
    <div>
        <LayoutContent v-loading="loading" :title="$t('github.projects')">
            <template #prompt>
                <el-alert v-if="setting && !setting.gitInstalled" type="warning" :closable="false">
                    <template #title>{{ $t('github.gitMissing') }}</template>
                </el-alert>
            </template>
            <template #leftToolBar>
                <el-button v-permission v-node-admin type="primary" @click="onCreate()">
                    {{ $t('commons.button.create') }}
                </el-button>
            </template>
            <template #rightToolBar>
                <TableSearch @search="search()" v-model:searchName="searchName" />
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
                    <el-table-column :label="$t('github.name')" prop="name" min-width="140" show-overflow-tooltip>
                        <template #default="{ row }">
                            <div class="flex flex-col">
                                <span>{{ row.name }}</span>
                                <el-link
                                    type="info"
                                    :underline="false"
                                    class="text-xs justify-start"
                                    :href="'https://github.com/' + row.repo"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    {{ row.repo }}
                                </el-link>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.branch')" prop="branch" min-width="90" show-overflow-tooltip />
                    <el-table-column
                        :label="$t('github.projectPath')"
                        prop="projectPath"
                        min-width="160"
                        show-overflow-tooltip
                    >
                        <template #default="{ row }">
                            <el-button text type="primary" @click="routerToFileWithPath(row.projectPath)">
                                {{ row.projectPath }}
                            </el-button>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.webhook')" min-width="100">
                        <template #default="{ row }">
                            <el-tag :type="row.webhookEnabled ? 'success' : 'info'">
                                {{ row.webhookEnabled ? $t('github.on') : $t('github.off') }}
                            </el-tag>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.status')" min-width="110">
                        <template #default="{ row }">
                            <el-tag v-if="row.running" type="warning">{{ $t('github.running') }}</el-tag>
                            <el-tag v-else-if="row.lastStatus" :type="statusType(row.lastStatus)">
                                {{ statusLabel(row.lastStatus) }}
                            </el-tag>
                            <span v-else>{{ $t('github.never') }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.lastRun')" min-width="150">
                        <template #default="{ row }">
                            <div class="flex flex-col">
                                <span>{{ formatTime(row.lastRunAt) }}</span>
                                <span class="text-xs text-gray-400" v-if="row.lastCommit">
                                    {{ shortCommit(row.lastCommit) }}
                                </span>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.enabled')" min-width="110">
                        <template #default="{ row }">
                            <el-switch v-model="row.enabled" v-permission v-node-admin @change="onChangeStatus(row)" />
                        </template>
                    </el-table-column>
                    <fu-table-operations
                        width="260px"
                        :buttons="buttons"
                        :ellipsis="3"
                        :label="$t('commons.table.operate')"
                        fix
                    />
                </ComplexTable>
            </template>
        </LayoutContent>

        <OpDialog ref="opRef" @search="search" @submit="onSubmitDelete()" />
        <CreateDialog ref="createRef" @search="search" />
        <EditDrawer ref="editRef" @search="search" />
        <WebhookDialog ref="webhookRef" @search="search" />
        <DeployDialog ref="deployRef" @search="search" />
    </div>
</template>

<script lang="ts" setup>
// fork: GitHub projects list.
import { onMounted, onUnmounted, reactive, ref } from 'vue';
import i18n from '@/lang';
import { ElMessageBox } from 'element-plus';
import { MsgSuccess } from '@/utils/message';
import { routerToFileWithPath } from '@/utils/router';
import { Github } from '@/api/interface/github';
import {
    cancelGithubDeploy,
    deleteGithubProject,
    loadGithubSetting,
    searchGithubProjects,
    updateGithubProjectStatus,
} from '@/api/modules/github';
import CreateDialog from '@/views/github/projects/create.vue';
import EditDrawer from '@/views/github/projects/edit.vue';
import WebhookDialog from '@/views/github/projects/webhook.vue';
import DeployDialog from '@/views/github/projects/deploy.vue';
import { formatTime, shortCommit, statusLabel, statusType } from '@/views/github/helper';
import { useRouter } from 'vue-router';

const router = useRouter();
const loading = ref(false);
const data = ref<Github.ProjectInfo[]>([]);
const searchName = ref('');
const setting = ref<Github.SettingInfo>();
const paginationConfig = reactive({
    currentPage: 1,
    pageSize: 20,
    total: 0,
});
const opRef = ref();
const createRef = ref();
const editRef = ref();
const webhookRef = ref();
const deployRef = ref();
const deleteID = ref(0);
let timer: ReturnType<typeof setInterval> | undefined;

const search = async () => {
    loading.value = true;
    try {
        const res = await searchGithubProjects({
            info: searchName.value,
            standalone: true,
            page: paginationConfig.currentPage,
            pageSize: paginationConfig.pageSize,
        });
        data.value = res.data.items || [];
        paginationConfig.total = res.data.total;
    } finally {
        loading.value = false;
    }
    loadGithubSetting()
        .then((res) => {
            setting.value = res.data;
        })
        .catch(() => {});
};

const refreshQuietly = async () => {
    if (!data.value.some((item) => item.running)) {
        return;
    }
    const res = await searchGithubProjects({
        info: searchName.value,
        standalone: true,
        page: paginationConfig.currentPage,
        pageSize: paginationConfig.pageSize,
    });
    data.value = res.data.items || [];
};

const onCreate = () => {
    createRef.value.acceptParams();
};
const onEdit = (row: Github.ProjectInfo) => {
    editRef.value.acceptParams(row);
};

const onChangeStatus = async (row: Github.ProjectInfo) => {
    try {
        await updateGithubProjectStatus(row.id, row.enabled);
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
    } catch {
        row.enabled = !row.enabled;
    }
};

const onDeploy = (row: Github.ProjectInfo) => {
    deployRef.value.acceptParams(row.id, row.name);
};

const onCancel = (row: Github.ProjectInfo) => {
    ElMessageBox.confirm(i18n.global.t('github.cancelHelper', [row.name]), i18n.global.t('github.cancelRun'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await cancelGithubDeploy(row.id);
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        search();
    });
};

const onDelete = (row: Github.ProjectInfo) => {
    deleteID.value = row.id;
    opRef.value.acceptParams({
        title: i18n.global.t('commons.button.delete'),
        names: [row.name],
        msg: i18n.global.t('github.deleteHelper', [row.name]),
        api: null,
        params: null,
    });
};

const onSubmitDelete = async () => {
    loading.value = true;
    try {
        await deleteGithubProject(deleteID.value);
        MsgSuccess(i18n.global.t('commons.msg.deleteSuccess'));
    } finally {
        loading.value = false;
    }
    search();
};

const buttons = [
    {
        label: i18n.global.t('github.deployNow'),
        permission: true,
        nodeAdmin: true,
        disabled: (row: Github.ProjectInfo) => row.running,
        click: onDeploy,
    },
    {
        label: i18n.global.t('github.webhook'),
        disabled: (row: Github.ProjectInfo) => !row.webhookEnabled,
        click: (row: Github.ProjectInfo) => webhookRef.value.acceptParams(row.id),
    },
    {
        label: i18n.global.t('github.viewLog'),
        click: (row: Github.ProjectInfo) => router.push({ path: '/github/logs', query: { projectID: String(row.id) } }),
    },
    {
        label: i18n.global.t('github.cancelRun'),
        permission: true,
        nodeAdmin: true,
        disabled: (row: Github.ProjectInfo) => !row.running,
        click: onCancel,
    },
    {
        label: i18n.global.t('commons.button.edit'),
        permission: true,
        nodeAdmin: true,
        click: onEdit,
    },
    {
        label: i18n.global.t('commons.button.delete'),
        permission: true,
        nodeAdmin: true,
        click: onDelete,
    },
];

onMounted(() => {
    search();
    timer = setInterval(() => {
        refreshQuietly().catch(() => {});
    }, 5000);
});

onUnmounted(() => {
    if (timer) {
        clearInterval(timer);
    }
});
</script>
