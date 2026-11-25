<template>
    <DrawerPro v-model="visible" :header="$t('github.scripts')" :resource="projectName" size="large" @close="onClose">
        <div v-loading="loading">
            <div class="mb-3 flex items-center justify-between">
                <span class="input-help">{{ $t('github.scriptsHelper') }}</span>
                <el-button v-permission v-node-admin type="primary" @click="onOpenForm()">
                    {{ $t('github.addScript') }}
                </el-button>
            </div>
            <el-table :data="scripts" row-key="id">
                <el-table-column :label="$t('github.scriptName')" prop="name" min-width="140" show-overflow-tooltip>
                    <template #default="{ row }">
                        <span>{{ row.name }}</span>
                        <el-tag v-if="row.isDefault" size="small" class="ml-2">{{ $t('github.defaultScript') }}</el-tag>
                        <el-tag v-if="row.linked" size="small" type="success" class="ml-1">
                            {{ $t('github.linkedScript') }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column :label="$t('github.scriptRunAs')" min-width="110">
                    <template #default="{ row }">{{ row.runAsUser || 'root' }}</template>
                </el-table-column>
                <el-table-column :label="$t('commons.table.operate')" width="150" fixed="right">
                    <template #default="{ row }">
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
        </div>
        <template #footer>
            <el-button @click="visible = false">{{ $t('commons.button.close') }}</el-button>
        </template>
    </DrawerPro>

    <DialogPro v-model="formVisible" :title="form.id ? $t('github.editScript') : $t('github.addScript')" size="large">
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
            <span class="input-help mb-2">{{ $t('github.scriptRunAsHelper') }}</span>
            <el-form-item :label="$t('github.scriptContent')" prop="content">
                <el-input v-model="form.content" type="textarea" :rows="12" class="font-mono" />
                <span class="input-help">{{ $t('github.deployCommandHelper') }}</span>
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button @click="formVisible = false">{{ $t('commons.button.cancel') }}</el-button>
            <el-button type="primary" :disabled="saving" @click="onSave">{{ $t('commons.button.confirm') }}</el-button>
        </template>
    </DialogPro>
</template>

<script lang="ts" setup>
// fork: deploy scripts of a project. One is the default, the webhook is linked to exactly one.
import { reactive, ref } from 'vue';
import { ElForm, ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import {
    createGithubScript,
    deleteGithubScript,
    listGithubScripts,
    listGithubSystemUsers,
    loadGithubDefaultScript,
    updateGithubScript,
} from '@/api/modules/github';

type FormInstance = InstanceType<typeof ElForm>;

const emit = defineEmits<{ (e: 'changed'): void }>();

const visible = ref(false);
const formVisible = ref(false);
const loading = ref(false);
const saving = ref(false);
const projectID = ref(0);
const projectName = ref('');
const scripts = ref<Github.ScriptInfo[]>([]);
const users = ref<string[]>([]);
const formRef = ref<FormInstance>();

const emptyForm = (): Github.ScriptOperate => ({ id: 0, projectID: 0, name: '', content: '', runAsUser: '' });
const form = reactive<Github.ScriptOperate>(emptyForm());
const rules = reactive({
    name: [Rules.requiredInput],
    runAsUser: [Rules.requiredSelect],
});

const load = async () => {
    loading.value = true;
    try {
        const res = await listGithubScripts(projectID.value);
        scripts.value = res.data || [];
    } finally {
        loading.value = false;
    }
};

const acceptParams = (id: number, name: string) => {
    projectID.value = id;
    projectName.value = name;
    scripts.value = [];
    visible.value = true;
    load().catch(() => {});
    listGithubSystemUsers()
        .then((res) => {
            users.value = res.data || [];
        })
        .catch(() => {});
};

const onOpenForm = async (row?: Github.ScriptInfo) => {
    Object.assign(form, emptyForm(), { projectID: projectID.value });
    if (row) {
        Object.assign(form, { id: row.id, name: row.name, content: row.content, runAsUser: row.runAsUser || 'root' });
    } else {
        // The backend fills in the folder and container of the project.
        const res = await loadGithubDefaultScript(projectID.value).catch(() => undefined);
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
            emit('changed');
        } finally {
            saving.value = false;
        }
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
        emit('changed');
    });
};

const onClose = () => {
    visible.value = false;
};

defineExpose({ acceptParams });
</script>
