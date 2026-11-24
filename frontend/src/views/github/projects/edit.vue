<template>
    <DrawerPro v-model="drawerVisible" :header="$t('commons.button.edit')" :resource="form.name" size="large">
        <el-form ref="formRef" label-position="top" :model="form" :rules="rules" v-loading="loading" @submit.prevent>
            <SourceFields :source="source" :repo="repo" :errors="errors" :token-stored="tokenStored" test-button />

            <el-form-item :label="$t('github.name')" prop="name">
                <el-input v-model.trim="form.name" />
            </el-form-item>
            <el-form-item :label="$t('github.projectPath')" prop="projectPath">
                <el-input v-model.trim="form.projectPath">
                    <template #prepend>
                        <el-button icon="Folder" @click="fileRef.acceptParams({ dir: true })" />
                    </template>
                </el-input>
                <span class="input-help">{{ $t('github.projectPathHelper') }}</span>
            </el-form-item>
            <el-form-item :label="$t('github.timeout')" prop="timeout">
                <el-input-number v-model="form.timeout" :min="30" :max="86400" :step="60" />
            </el-form-item>
            <el-form-item :label="$t('github.deploymentRecords')">
                <el-select v-model="form.keepRecords" class="w-32">
                    <el-option v-for="item in [5, 10, 20, 30]" :key="item" :value="item" :label="String(item)" />
                </el-select>
                <span class="input-help">{{ $t('github.deploymentRecordsHelper') }}</span>
            </el-form-item>
            <el-form-item :label="$t('github.ignorePaths')">
                <el-input v-model="form.ignorePaths" type="textarea" :rows="4" class="font-mono" />
                <span class="input-help">{{ $t('github.ignorePathsHelper') }}</span>
            </el-form-item>

            <el-divider content-position="left">{{ $t('github.webhook') }}</el-divider>
            <el-form-item :label="$t('github.webhookMode')">
                <el-switch v-model="webhookOn" />
                <span class="input-help ml-2">
                    {{ webhookOn ? $t('github.webhookOn') : $t('github.webhookOff') }}
                </span>
            </el-form-item>
            <el-form-item v-if="webhookOn" :label="$t('github.webhookScript')">
                <div class="flex w-full gap-2">
                    <el-select v-model="webhookScriptID" class="flex-1">
                        <el-option v-for="item in scripts" :key="item.id" :value="item.id" :label="item.name">
                            <span>{{ item.name }}</span>
                            <el-tag v-if="item.isDefault" size="small" class="ml-2">
                                {{ $t('github.defaultScript') }}
                            </el-tag>
                        </el-option>
                    </el-select>
                    <el-button @click="scriptsRef.acceptParams(form.id, form.name)">
                        {{ $t('github.manageScripts') }}
                    </el-button>
                </div>
                <span class="input-help">{{ $t('github.webhookScriptHelper') }}</span>
            </el-form-item>
            <el-form-item>
                <el-button
                    v-permission
                    v-node-admin
                    type="primary"
                    plain
                    :loading="savingWebhook"
                    @click="onSaveWebhook"
                >
                    {{ $t('github.saveWebhook') }}
                </el-button>
                <el-button v-if="!webhookOn" @click="scriptsRef.acceptParams(form.id, form.name)">
                    {{ $t('github.manageScripts') }}
                </el-button>
                <el-button v-if="webhookOn && info?.webhookEnabled" @click="webhookRef.acceptParams(form.id)">
                    {{ $t('github.webhookDetails') }}
                </el-button>
            </el-form-item>
            <span class="input-help">{{ $t('github.boundRestartHelper') }}</span>
        </el-form>
        <template #footer>
            <span class="dialog-footer">
                <el-button @click="drawerVisible = false">{{ $t('commons.button.cancel') }}</el-button>
                <el-button v-permission v-node-admin :disabled="loading" type="primary" @click="onSubmit">
                    {{ $t('commons.button.confirm') }}
                </el-button>
            </span>
        </template>
    </DrawerPro>
    <FileList ref="fileRef" @choose="(path: string) => (form.projectPath = path)" />
    <ScriptsDialog ref="scriptsRef" @changed="loadScripts" />
    <WebhookDialog ref="webhookRef" @search="emit('search')" />
</template>

<script lang="ts" setup>
// fork: edit a GitHub project. Everything the create form leaves out is here: timeout, deployment
// records, ignored paths and the webhook.
import { computed, reactive, ref } from 'vue';
import { ElForm } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgSuccess } from '@/utils/message';
import FileList from '@/components/file-list/index.vue';
import ScriptsDialog from '@/views/github/projects/scripts.vue';
import WebhookDialog from '@/views/github/projects/webhook.vue';
import SourceFields from '@/views/github/components/source-fields/index.vue';
import {
    buildCloneURL,
    emptySource,
    keyModeOf,
    repoFromInput,
    sourceError,
    useGitRepo,
} from '@/views/github/composables/use-git-repo';
import { Github } from '@/api/interface/github';
import { configureGithubWebhook, listGithubScripts, updateGithubProject } from '@/api/modules/github';

type FormInstance = InstanceType<typeof ElForm>;

const emit = defineEmits<{ (e: 'search'): void }>();

const drawerVisible = ref(false);
const loading = ref(false);
const savingWebhook = ref(false);
const formRef = ref<FormInstance>();
const fileRef = ref();
const scriptsRef = ref();
const webhookRef = ref();
const scripts = ref<Github.ScriptInfo[]>([]);
const info = ref<Github.ProjectInfo>();
const webhookOn = ref(false);
const webhookScriptID = ref(0);
const projectID = ref<number | undefined>();
const tokenStored = ref(false);
const errors = reactive({ url: '', token: '', key: '', branch: '' });

const source = reactive<Github.Source>(emptySource());
const repo = useGitRepo(source, { keepBranch: true, projectID: projectID, tokenStored: tokenStored });

const emptyForm = () => ({
    id: 0,
    name: '',
    projectPath: '',
    timeout: 600,
    keepRecords: 5,
    ignorePaths: '',
    enabled: true,
    updateStrategy: 'ff-only' as Github.UpdateStrategy,
});
const form = reactive(emptyForm());

const checkPath = (_rule: any, value: string, callback: any) => {
    if (!value || !value.startsWith('/') || value === '/') {
        callback(new Error(i18n.global.t('github.projectPathHelper')));
        return;
    }
    callback();
};
const rules = reactive({
    name: [Rules.requiredInput],
    projectPath: [{ validator: checkPath, trigger: 'blur', required: true }],
});

const hasLinked = computed(() => !!(info.value?.websiteID || info.value?.runtimeID));

const loadScripts = async () => {
    if (!form.id) return;
    const res = await listGithubScripts(form.id);
    scripts.value = res.data || [];
    if (!scripts.value.some((item) => item.id === webhookScriptID.value)) {
        webhookScriptID.value = scripts.value.find((item) => item.linked)?.id || scripts.value[0]?.id || 0;
    }
};

const acceptParams = (row: Github.ProjectInfo) => {
    Object.assign(form, emptyForm(), {
        id: row.id,
        name: row.name,
        projectPath: row.projectPath,
        timeout: row.timeout,
        keepRecords: row.keepRecords || 5,
        ignorePaths: row.ignorePaths || '',
        enabled: row.enabled,
        updateStrategy: row.updateStrategy || 'ff-only',
    });
    info.value = row;
    projectID.value = row.id;
    tokenStored.value = row.keyMode === 'token' && row.hasToken;
    Object.assign(source, {
        cloneURL: row.cloneURL,
        branch: row.branch,
        authMode: row.keyMode === 'token' ? 'token' : row.keyMode === 'none' ? 'public' : 'ssh',
        keyID: row.keyMode === 'shared' ? 0 : row.keyID,
        tokenUser: row.tokenUser || '',
        token: '',
    });
    errors.url = errors.token = errors.key = errors.branch = '';
    repo.onUrlChange();
    scripts.value = [];
    webhookOn.value = row.webhookEnabled;
    webhookScriptID.value = row.webhookScriptID;
    drawerVisible.value = true;
    repo.loadKeys().catch(() => {});
    loadScripts().catch(() => {});
    if (row.keyMode !== 'none' || !row.cloneURL.startsWith('git@')) {
        repo.check(true).catch(() => {});
    }
};

const onSubmit = async () => {
    const formOK = await formRef.value?.validate().catch(() => false);
    if (!formOK) return;
    errors.url = errors.token = errors.key = errors.branch = '';
    const message = sourceError(source, tokenStored.value);
    if (message) {
        if (!source.branch.trim()) errors.branch = message;
        else if (source.authMode === 'token' && !source.token.trim() && !tokenStored.value) errors.token = message;
        else errors.url = message;
        return;
    }
    loading.value = true;
    try {
        const cloneURL = buildCloneURL(source);
        const params: Github.ProjectOperate = {
            id: form.id,
            name: form.name,
            cloneURL: cloneURL,
            repo: repoFromInput(cloneURL),
            branch: source.branch,
            projectPath: form.projectPath,
            updateStrategy: form.updateStrategy,
            timeout: form.timeout,
            keyMode: keyModeOf(source),
            keyID: source.authMode === 'ssh' ? source.keyID : 0,
            tokenUser: source.authMode === 'token' ? source.tokenUser : '',
            token: source.authMode === 'token' ? source.token : '',
            keepRecords: form.keepRecords,
            ignorePaths: form.ignorePaths,
            enabled: form.enabled,
        };
        await updateGithubProject(params);
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        drawerVisible.value = false;
        emit('search');
    } finally {
        loading.value = false;
    }
};

// The webhook has its own save: switching it on creates the URL, shown right after.
const onSaveWebhook = async () => {
    savingWebhook.value = true;
    try {
        await configureGithubWebhook({
            id: form.id,
            enabled: webhookOn.value,
            scriptID: webhookScriptID.value,
        });
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        if (info.value) {
            info.value.webhookEnabled = webhookOn.value;
        }
        emit('search');
        if (webhookOn.value) {
            webhookRef.value.acceptParams(form.id);
        }
    } finally {
        savingWebhook.value = false;
    }
};

defineExpose({ acceptParams, hasLinked });
</script>
