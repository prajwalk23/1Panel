<template>
    <DrawerPro
        v-model="drawerVisible"
        :header="title"
        :resource="form.id ? form.name : ''"
        @close="handleClose"
        size="large"
    >
        <el-form ref="formRef" label-position="top" :model="form" :rules="rules" v-loading="loading">
            <el-form-item :label="$t('github.keyMode')" prop="keyID">
                <div class="flex w-full gap-2">
                    <el-select v-model="form.keyID" class="flex-1" @change="onKeyChange">
                        <el-option :value="0" :label="$t('github.keyNoneOption')" />
                        <el-option v-for="item in keys" :key="item.id" :value="item.id" :label="item.name" />
                    </el-select>
                    <el-button @click="showKeyCreate = !showKeyCreate">{{ $t('github.createKey') }}</el-button>
                </div>
                <div v-if="showKeyCreate" class="mt-2 flex w-full gap-2">
                    <el-input v-model.trim="newKeyName" :placeholder="$t('github.keyNamePlaceholder')" />
                    <el-button type="primary" :disabled="!newKeyName" :loading="creatingKey" @click="onCreateKey">
                        {{ $t('commons.button.create') }}
                    </el-button>
                </div>
                <template v-if="selectedKey">
                    <span class="input-help">{{ $t('github.keyProjectHelper') }}</span>
                    <div class="mt-1 flex w-full items-start gap-1">
                        <el-input
                            :model-value="selectedKey.publicKey"
                            type="textarea"
                            :rows="3"
                            readonly
                            class="font-mono"
                        />
                        <CopyButton :content="selectedKey.publicKey" />
                    </div>
                    <el-link
                        v-if="githubRepoName"
                        type="primary"
                        :underline="false"
                        :href="'https://github.com/' + githubRepoName + '/settings/keys/new'"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="mt-1"
                    >
                        {{ $t('github.openDeployKeys') }}
                    </el-link>
                </template>
                <span v-else class="input-help">{{ $t('github.keyNoneHelper') }}</span>
            </el-form-item>

            <el-form-item :label="$t('github.repo')" prop="repoInput">
                <div class="flex w-full gap-2">
                    <el-input
                        v-model.trim="repoInput"
                        :placeholder="$t('github.repoPlaceholder')"
                        @change="onRepoChange"
                    />
                    <el-button :loading="checking" @click="onCheck">{{ $t('github.checkRepo') }}</el-button>
                </div>
                <span class="input-help">{{ $t('github.repoHelper') }}</span>
                <el-alert
                    v-if="checkResult"
                    :type="checkResult.ok ? 'success' : 'error'"
                    :closable="false"
                    class="mt-2"
                >
                    <template #title>
                        <span v-if="checkResult.ok">{{ $t('github.repoOK', [checkResult.branches.length]) }}</span>
                        <template v-else>
                            <div>{{ $t('github.repoFail') }}</div>
                            <div class="text-xs break-all">{{ checkResult.message }}</div>
                            <div v-if="selectedKey" class="text-xs">{{ $t('github.repoNeedsKey') }}</div>
                        </template>
                    </template>
                </el-alert>
            </el-form-item>

            <el-form-item :label="$t('github.branch')" prop="branch">
                <div class="flex w-full gap-2">
                    <el-select
                        v-model="form.branch"
                        filterable
                        allow-create
                        default-first-option
                        class="flex-1"
                        :placeholder="$t('github.repoCheckFirst')"
                    >
                        <el-option v-for="item in branchOptions" :key="item" :value="item" :label="item" />
                    </el-select>
                    <el-tooltip :content="$t('github.refreshBranches')" placement="top">
                        <el-button :loading="checking" icon="Refresh" @click="onCheck" />
                    </el-tooltip>
                </div>
            </el-form-item>

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

            <template v-if="form.id">
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
            </template>
        </el-form>
        <template #footer>
            <span class="dialog-footer">
                <el-button @click="drawerVisible = false">{{ $t('commons.button.cancel') }}</el-button>
                <el-button v-permission v-node-admin :disabled="loading" type="primary" @click="onSubmit(formRef)">
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
// fork: create or edit a GitHub project. The webhook is configured in edit only.
import { computed, reactive, ref } from 'vue';
import { ElForm } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgError, MsgSuccess } from '@/utils/message';
import FileList from '@/components/file-list/index.vue';
import ScriptsDialog from '@/views/github/projects/scripts.vue';
import WebhookDialog from '@/views/github/projects/webhook.vue';
import { Github } from '@/api/interface/github';
import {
    checkGithubRepo,
    configureGithubWebhook,
    createGithubKey,
    createGithubProject,
    listGithubKeyOptions,
    listGithubScripts,
    updateGithubProject,
} from '@/api/modules/github';

type FormInstance = InstanceType<typeof ElForm>;

const emit = defineEmits<{ (e: 'search'): void }>();

const drawerVisible = ref(false);
const loading = ref(false);
const checking = ref(false);
const creatingKey = ref(false);
const savingWebhook = ref(false);
const formRef = ref<FormInstance>();
const fileRef = ref();
const scriptsRef = ref();
const webhookRef = ref();
const keys = ref<Github.KeyOption[]>([]);
const scripts = ref<Github.ScriptInfo[]>([]);
const branches = ref<string[]>([]);
const checkResult = ref<Github.RepoCheckInfo>();
const repoInput = ref('');
const showKeyCreate = ref(false);
const newKeyName = ref('');
const info = ref<Github.ProjectInfo>();
const webhookOn = ref(false);
const webhookScriptID = ref(0);

const emptyForm = (): Github.ProjectOperate => ({
    id: 0,
    name: '',
    cloneURL: '',
    repo: '',
    branch: '',
    projectPath: '',
    updateStrategy: 'ff-only',
    timeout: 600,
    keyMode: 'none',
    keyID: 0,
    enabled: true,
});
const form = reactive<Github.ProjectOperate>(emptyForm());

const title = computed(() => (form.id ? i18n.global.t('commons.button.edit') : i18n.global.t('commons.button.create')));
const selectedKey = computed(() => keys.value.find((item) => item.id === form.keyID));
const branchOptions = computed(() => {
    const list = [...branches.value];
    if (form.branch && !list.includes(form.branch)) {
        list.unshift(form.branch);
    }
    return list;
});

// owner/repo of whatever was typed: owner/repo, git@github.com:owner/repo.git or an https URL.
const repoFromInput = (value: string): string => {
    let text = value
        .trim()
        .replace(/\/$/, '')
        .replace(/\.git$/, '');
    if (text.startsWith('git@')) {
        text = text.substring(text.indexOf(':') + 1);
    } else if (text.includes('://')) {
        text = text.substring(text.indexOf('://') + 3);
        text = text.substring(text.indexOf('/') + 1);
    }
    return /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(text) ? text : '';
};
const githubRepoName = computed(() => repoFromInput(repoInput.value));

const buildCloneURL = (): string => {
    const value = repoInput.value.trim();
    if (value.includes('://') || value.startsWith('git@')) {
        return value;
    }
    const name = repoFromInput(value);
    if (!name) {
        return '';
    }
    return form.keyID ? `git@github.com:${name}.git` : `https://github.com/${name}.git`;
};

const loadKeys = async () => {
    const res = await listGithubKeyOptions();
    keys.value = res.data || [];
};
const loadScripts = async () => {
    if (!form.id) return;
    const res = await listGithubScripts(form.id);
    scripts.value = res.data || [];
    if (!scripts.value.some((item) => item.id === webhookScriptID.value)) {
        webhookScriptID.value = scripts.value.find((item) => item.linked)?.id || scripts.value[0]?.id || 0;
    }
};

const onKeyChange = () => {
    checkResult.value = undefined;
};
const onRepoChange = () => {
    checkResult.value = undefined;
    branches.value = [];
};

const onCheck = async () => {
    const cloneURL = buildCloneURL();
    if (!cloneURL) {
        MsgError(i18n.global.t('github.repoHelper'));
        return;
    }
    checking.value = true;
    try {
        const res = await checkGithubRepo({
            cloneURL: cloneURL,
            keyMode: form.keyID ? 'custom' : 'none',
            keyID: form.keyID,
        });
        checkResult.value = res.data;
        branches.value = res.data.branches || [];
        if (res.data.ok && (!form.branch || (!form.id && !branches.value.includes(form.branch)))) {
            form.branch = res.data.defaultBranch || branches.value[0] || '';
        }
    } finally {
        checking.value = false;
    }
};

const onCreateKey = async () => {
    creatingKey.value = true;
    try {
        const res = await createGithubKey(newKeyName.value, 'shared');
        await loadKeys();
        form.keyID = res.data.id;
        newKeyName.value = '';
        showKeyCreate.value = false;
        checkResult.value = undefined;
        MsgSuccess(i18n.global.t('github.keyCreated'));
    } finally {
        creatingKey.value = false;
    }
};

const checkPath = (rule: any, value: string, callback: any) => {
    if (!value || !value.startsWith('/') || value === '/') {
        callback(new Error(i18n.global.t('github.projectPathHelper')));
        return;
    }
    callback();
};
const checkRepoInput = (rule: any, value: string, callback: any) => {
    if (!buildCloneURL() || /[\s'"`;|&$<>\\]/.test(repoInput.value)) {
        callback(new Error(i18n.global.t('github.repoHelper')));
        return;
    }
    callback();
};

const rules = reactive({
    name: [Rules.requiredInput],
    repoInput: [{ validator: checkRepoInput, trigger: 'blur', required: true }],
    branch: [Rules.requiredInput, Rules.noSpace],
    projectPath: [{ validator: checkPath, trigger: 'blur', required: true }],
});

const acceptParams = (row?: Github.ProjectInfo) => {
    Object.assign(form, emptyForm());
    info.value = row;
    checkResult.value = undefined;
    branches.value = [];
    scripts.value = [];
    showKeyCreate.value = false;
    newKeyName.value = '';
    repoInput.value = '';
    webhookOn.value = false;
    webhookScriptID.value = 0;
    if (row) {
        Object.assign(form, {
            id: row.id,
            name: row.name,
            cloneURL: row.cloneURL,
            repo: row.repo,
            branch: row.branch,
            projectPath: row.projectPath,
            updateStrategy: row.updateStrategy,
            timeout: row.timeout,
            keyID: row.keyMode === 'none' ? 0 : row.keyID,
            enabled: row.enabled,
        });
        repoInput.value = row.cloneURL;
        webhookOn.value = row.webhookEnabled;
        webhookScriptID.value = row.webhookScriptID;
    }
    drawerVisible.value = true;
    loadKeys().catch(() => {});
    if (row) {
        loadScripts().catch(() => {});
        if (row.keyMode !== 'none' || !row.cloneURL.startsWith('git@')) {
            onCheck().catch(() => {});
        }
    }
};

const handleClose = () => {
    drawerVisible.value = false;
    formRef.value?.resetFields();
};

const onSubmit = async (formEl: FormInstance | undefined) => {
    if (!formEl) return;
    await formEl.validate(async (valid) => {
        if (!valid) return;
        loading.value = true;
        try {
            const params: Github.ProjectOperate = { ...form };
            params.cloneURL = buildCloneURL();
            params.repo = repoFromInput(params.cloneURL) || repoFromInput(repoInput.value);
            params.keyMode = params.keyID ? 'custom' : 'none';
            if (params.id) {
                await updateGithubProject(params);
            } else {
                await createGithubProject(params);
            }
            MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
            drawerVisible.value = false;
            emit('search');
        } finally {
            loading.value = false;
        }
    });
};

// The webhook has its own save: switching it on returns the secret, which is shown once.
const onSaveWebhook = async () => {
    savingWebhook.value = true;
    try {
        const res = await configureGithubWebhook({
            id: form.id as number,
            enabled: webhookOn.value,
            scriptID: webhookScriptID.value,
        });
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        if (info.value) {
            info.value.webhookEnabled = webhookOn.value;
        }
        emit('search');
        if (webhookOn.value) {
            webhookRef.value.acceptParams(form.id, res.data.secret ? res.data : undefined);
        }
    } finally {
        savingWebhook.value = false;
    }
};

defineExpose({ acceptParams });
</script>
