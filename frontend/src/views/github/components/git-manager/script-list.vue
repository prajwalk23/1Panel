<template>
    <div v-loading="loading">
        <div class="mb-3 flex items-center justify-between">
            <div class="text-base font-bold">{{ $t('github.deploymentScripts') }}</div>
            <el-button v-permission v-node-admin type="primary" @click="onOpenForm()">
                {{ $t('github.addScript') }}
            </el-button>
        </div>
        <el-alert type="info" :closable="false" class="mb-3">
            <template #title>{{ $t('github.scriptsOptionalTitle') }}</template>
            {{ $t('github.scriptsOptionalHelper') }}
        </el-alert>
        <el-table :data="scripts" row-key="id">
            <el-table-column :label="$t('github.scriptColumn')" prop="name" min-width="120" show-overflow-tooltip>
                <template #default="{ row }">
                    <span>{{ row.name }}</span>
                    <el-tag v-if="row.linked" size="small" type="success" class="ml-1">
                        {{ $t('github.linkedScript') }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column :label="$t('github.commandsColumn')" min-width="220" show-overflow-tooltip>
                <template #default="{ row }">
                    <span class="font-mono text-xs">{{ commandsOf(row.content) }}</span>
                </template>
            </el-table-column>
            <el-table-column :label="$t('commons.table.operate')" width="190" align="right">
                <template #default="{ row }">
                    <el-button link type="primary" v-permission v-node-admin @click="onRun(row)">
                        {{ $t('github.runScript') }}
                    </el-button>
                    <el-button link type="primary" v-permission v-node-admin @click="onOpenForm(row)">
                        {{ $t('commons.button.edit') }}
                    </el-button>
                    <el-button
                        link
                        type="primary"
                        v-permission
                        v-node-admin
                        :disabled="row.isDefault"
                        @click="onDelete(row)"
                    >
                        {{ $t('commons.button.delete') }}
                    </el-button>
                </template>
            </el-table-column>
        </el-table>

        <DialogPro
            v-model="formVisible"
            :title="form.id ? $t('github.editScript') : $t('github.addScript')"
            size="large"
        >
            <el-form ref="formRef" label-position="top" :model="form" :rules="rules">
                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-form-item :label="$t('github.scriptName')" prop="name">
                            <el-input v-model.trim="form.name" />
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item :label="$t('github.scriptRunAs')" prop="runAsUser">
                            <el-select v-model="form.runAsUser" class="w-full">
                                <el-option value="root" :label="$t('github.rootUser')" />
                                <el-option v-for="item in users" :key="item" :value="item" :label="item" />
                            </el-select>
                        </el-form-item>
                    </el-col>
                </el-row>
                <el-form-item :label="$t('github.scriptContent')" prop="content">
                    <div class="w-full">
                        <CodemirrorPro v-model="form.content" mode="shell" :minHeight="260" :heightDiff="520" />
                    </div>
                    <span class="input-help">{{ $t('github.deployCommandHelper') }}</span>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="formVisible = false">{{ $t('commons.button.cancel') }}</el-button>
                <el-button type="primary" :disabled="saving" @click="onSave">
                    {{ $t('commons.button.confirm') }}
                </el-button>
            </template>
        </DialogPro>
    </div>
</template>

<script lang="ts" setup>
// fork: Script tab of the Git Manager. Scripts are optional: deploy, versions and rollback work without one.
import { onMounted, reactive, ref } from 'vue';
import { ElForm, ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import CodemirrorPro from '@/components/codemirror-pro/index.vue';
import {
    createGithubScript,
    deleteGithubScript,
    deployGithubProject,
    listGithubScripts,
    listGithubSystemUsers,
    loadGithubDefaultScript,
    updateGithubScript,
} from '@/api/modules/github';

type FormInstance = InstanceType<typeof ElForm>;

const props = defineProps({
    projectId: {
        type: Number,
        required: true,
    },
    projectName: {
        type: String,
        default: '',
    },
});

const loading = ref(false);
const saving = ref(false);
const formVisible = ref(false);
const scripts = ref<Github.ScriptInfo[]>([]);
const users = ref<string[]>([]);
const formRef = ref<FormInstance>();

const emptyForm = (): Github.ScriptOperate => ({
    id: 0,
    projectID: props.projectId,
    name: '',
    content: '',
    runAsUser: '',
});
const form = reactive<Github.ScriptOperate>(emptyForm());
const rules = reactive({ name: [Rules.requiredInput], runAsUser: [Rules.requiredSelect] });

const commandsOf = (content: string): string =>
    content
        .split('\n')
        .map((line) => line.trim())
        .filter((line) => line && !line.startsWith('#'))
        .join(' ');

const load = async () => {
    loading.value = true;
    try {
        const res = await listGithubScripts(props.projectId);
        scripts.value = res.data || [];
    } finally {
        loading.value = false;
    }
};

const onOpenForm = async (row?: Github.ScriptInfo) => {
    Object.assign(form, emptyForm());
    if (row) {
        Object.assign(form, { id: row.id, name: row.name, content: row.content, runAsUser: row.runAsUser || 'root' });
    } else {
        // The backend fills in the folder and container of the project.
        const res = await loadGithubDefaultScript(props.projectId).catch(() => undefined);
        form.content = res?.data || '';
    }
    formRef.value?.clearValidate();
    formVisible.value = true;
};

const onSave = async () => {
    if (!formRef.value) return;
    await formRef.value.validate(async (valid) => {
        if (!valid) return;
        saving.value = true;
        try {
            if (form.id) {
                await updateGithubScript({ ...form });
            } else {
                await createGithubScript({ ...form });
            }
            MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
            formVisible.value = false;
            await load();
        } finally {
            saving.value = false;
        }
    });
};

const onRun = (row: Github.ScriptInfo) => {
    ElMessageBox.confirm(i18n.global.t('github.runScriptConfirm', [row.name]), i18n.global.t('github.runScript'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await deployGithubProject({ id: props.projectId, scriptID: row.id });
        MsgSuccess(i18n.global.t('github.deployStarted'));
    });
};

const onDelete = (row: Github.ScriptInfo) => {
    ElMessageBox.confirm(
        i18n.global.t('github.deleteScriptHelper', [row.name]),
        i18n.global.t('commons.button.delete'),
        {
            confirmButtonText: i18n.global.t('commons.button.confirm'),
            cancelButtonText: i18n.global.t('commons.button.cancel'),
            type: 'warning',
        },
    ).then(async () => {
        await deleteGithubScript(row.id);
        MsgSuccess(i18n.global.t('commons.msg.deleteSuccess'));
        await load();
    });
};

onMounted(() => {
    load().catch(() => {});
    listGithubSystemUsers()
        .then((res) => {
            users.value = res.data || [];
        })
        .catch(() => {});
});
</script>
