<template>
    <DrawerPro
        v-model="drawerVisible"
        :header="title"
        :resource="form.id ? form.name : ''"
        @close="handleClose"
        size="large"
    >
        <el-form ref="formRef" label-position="top" :model="form" :rules="rules" v-loading="loading">
            <el-form-item :label="$t('github.name')" prop="name">
                <el-input v-model.trim="form.name" />
            </el-form-item>
            <el-form-item :label="$t('github.cloneURL')" prop="cloneURL">
                <el-input
                    v-model.trim="form.cloneURL"
                    placeholder="git@github.com:owner/repo.git"
                    @change="onCloneURLChange"
                />
                <span class="input-help">{{ $t('github.cloneURLHelper') }}</span>
            </el-form-item>
            <el-row :gutter="20">
                <el-col :span="12">
                    <el-form-item :label="$t('github.repo')" prop="repo">
                        <el-input v-model.trim="form.repo" placeholder="owner/repo" />
                    </el-form-item>
                </el-col>
                <el-col :span="12">
                    <el-form-item :label="$t('github.branch')" prop="branch">
                        <el-input v-model.trim="form.branch" />
                    </el-form-item>
                </el-col>
            </el-row>
            <el-form-item :label="$t('github.projectPath')" prop="projectPath">
                <el-input v-model.trim="form.projectPath">
                    <template #prepend>
                        <el-button icon="Folder" @click="fileRef.acceptParams({ dir: true })" />
                    </template>
                </el-input>
                <span class="input-help">{{ $t('github.projectPathHelper') }}</span>
            </el-form-item>

            <el-form-item :label="$t('github.keyMode')" prop="keyMode">
                <el-select v-model="form.keyMode" class="w-full">
                    <el-option value="project" :label="$t('github.keyProject')" />
                    <el-option value="shared" :label="$t('github.keyShared')" />
                    <el-option value="custom" :label="$t('github.keyCustom')" />
                    <el-option value="none" :label="$t('github.keyNone')" />
                </el-select>
            </el-form-item>
            <el-form-item v-if="form.keyMode === 'custom'" :label="$t('github.selectKey')" prop="keyID">
                <el-select v-model="form.keyID" class="w-full">
                    <el-option
                        v-for="item in customKeys"
                        :key="item.id"
                        :value="item.id"
                        :label="item.name + ' (' + item.fingerprint + ')'"
                    />
                </el-select>
            </el-form-item>

            <el-form-item :label="$t('github.domainMode')" prop="domainMode">
                <el-radio-group v-model="form.domainMode">
                    <el-radio value="default">{{ $t('github.domainDefault') }}</el-radio>
                    <el-radio value="website">{{ $t('github.domainWebsite') }}</el-radio>
                    <el-radio value="custom">{{ $t('github.domainCustom') }}</el-radio>
                </el-radio-group>
            </el-form-item>
            <el-form-item v-if="form.domainMode === 'website'" :label="$t('github.domain')" prop="domainValue">
                <el-select v-model="domainSelection" filterable class="w-full">
                    <el-option
                        v-for="item in domains"
                        :key="domainValue(item.websiteID, item.domain)"
                        :value="domainValue(item.websiteID, item.domain)"
                        :label="item.domain"
                    >
                        <span>{{ item.domain }}</span>
                        <span class="text-xs text-gray-400 ml-2">{{ item.alias }}</span>
                        <el-tag size="small" class="ml-2" :type="item.https ? 'success' : 'warning'">
                            {{ item.https ? 'HTTPS' : 'HTTP' }}
                        </el-tag>
                    </el-option>
                </el-select>
            </el-form-item>
            <el-form-item v-if="form.domainMode === 'custom'" :label="$t('github.customURL')" prop="customURL">
                <el-input v-model.trim="form.customURL" placeholder="https://hooks.example.com" />
                <span class="input-help">{{ $t('github.customURLHelper') }}</span>
            </el-form-item>

            <el-form-item :label="$t('github.deployCommand')" prop="deployCommand">
                <el-input
                    v-model="form.deployCommand"
                    type="textarea"
                    :rows="6"
                    class="font-mono"
                    placeholder="npm ci && npm run build"
                />
                <span class="input-help">{{ $t('github.deployCommandHelper') }}</span>
            </el-form-item>
            <el-form-item :label="$t('github.updateStrategy')" prop="updateStrategy">
                <el-radio-group v-model="form.updateStrategy">
                    <el-radio value="ff-only">{{ $t('github.ffOnly') }}</el-radio>
                    <el-radio value="reset">{{ $t('github.reset') }}</el-radio>
                </el-radio-group>
                <span class="input-help">
                    {{ form.updateStrategy === 'reset' ? $t('github.resetHelper') : $t('github.ffOnlyHelper') }}
                </span>
            </el-form-item>
            <el-row :gutter="20">
                <el-col :span="12">
                    <el-form-item :label="$t('github.runAsUser')" prop="runAsUser">
                        <el-input v-model.trim="form.runAsUser" placeholder="www" />
                        <span class="input-help">{{ $t('github.runAsUserHelper') }}</span>
                    </el-form-item>
                </el-col>
                <el-col :span="12">
                    <el-form-item :label="$t('github.timeout')" prop="timeout">
                        <el-input-number v-model="form.timeout" :min="30" :max="86400" :step="60" />
                    </el-form-item>
                </el-col>
            </el-row>
            <el-form-item :label="$t('github.enabled')" prop="enabled">
                <el-switch v-model="form.enabled" />
            </el-form-item>
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
</template>

<script lang="ts" setup>
// fork: create or edit a GitHub project.
import { computed, reactive, ref } from 'vue';
import { ElForm } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgSuccess } from '@/utils/message';
import FileList from '@/components/file-list/index.vue';
import { Github } from '@/api/interface/github';
import {
    createGithubProject,
    listGithubDomains,
    listGithubKeyOptions,
    updateGithubProject,
} from '@/api/modules/github';
import { domainValue, parseDomainValue } from '@/views/github/helper';

type FormInstance = InstanceType<typeof ElForm>;

const emit = defineEmits<{
    (e: 'search'): void;
    (e: 'created', secret: Github.ProjectSecret): void;
}>();

const drawerVisible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();
const fileRef = ref();
const keys = ref<Github.KeyOption[]>([]);
const domains = ref<Github.DomainOption[]>([]);

const emptyForm = (): Github.ProjectOperate => ({
    id: 0,
    name: '',
    cloneURL: '',
    repo: '',
    branch: 'main',
    projectPath: '',
    deployCommand: '',
    updateStrategy: 'ff-only',
    runAsUser: '',
    timeout: 600,
    keyMode: 'project',
    keyID: 0,
    domainMode: 'default',
    websiteID: 0,
    domain: '',
    customURL: '',
    enabled: true,
});
const form = reactive<Github.ProjectOperate>(emptyForm());

const title = computed(() => (form.id ? i18n.global.t('commons.button.edit') : i18n.global.t('commons.button.create')));
const customKeys = computed(() => keys.value.filter((item) => item.type === 'custom'));
const domainSelection = computed({
    get: () => domainValue(form.websiteID, form.domain),
    set: (value: string) => {
        const parsed = parseDomainValue(value);
        form.websiteID = parsed.websiteID;
        form.domain = parsed.domain;
    },
});

const repoFromURL = (url: string): string => {
    let value = url
        .trim()
        .replace(/\/$/, '')
        .replace(/\.git$/, '');
    if (value.startsWith('git@')) {
        value = value.substring(value.indexOf(':') + 1);
    } else if (value.includes('://')) {
        value = value.substring(value.indexOf('://') + 3);
        value = value.substring(value.indexOf('/') + 1);
    } else {
        return '';
    }
    return /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(value) ? value : '';
};

const onCloneURLChange = () => {
    const repo = repoFromURL(form.cloneURL);
    if (repo) {
        form.repo = repo;
    }
    if (!form.id && form.cloneURL.startsWith('https://') && form.keyMode === 'project') {
        form.keyMode = 'none';
    }
};

const checkCloneURL = (rule: any, value: string, callback: any) => {
    if (!value || /[\s'"`;|&$<>\\]/.test(value) || !/^(git@[^:]+:|ssh:\/\/|https:\/\/)/.test(value)) {
        callback(new Error(i18n.global.t('github.cloneURLHelper')));
        return;
    }
    callback();
};
const checkPath = (rule: any, value: string, callback: any) => {
    if (!value || !value.startsWith('/') || value === '/') {
        callback(new Error(i18n.global.t('github.projectPathHelper')));
        return;
    }
    callback();
};
const checkDomain = (rule: any, value: any, callback: any) => {
    if (form.domainMode === 'website' && !form.websiteID) {
        callback(new Error(i18n.global.t('commons.rule.requiredSelect')));
        return;
    }
    callback();
};
const checkKey = (rule: any, value: any, callback: any) => {
    if (form.keyMode === 'custom' && !form.keyID) {
        callback(new Error(i18n.global.t('commons.rule.requiredSelect')));
        return;
    }
    callback();
};
const checkCustomURL = (rule: any, value: string, callback: any) => {
    if (form.domainMode === 'custom' && !/^https?:\/\/[^\s/]+/.test(value || '')) {
        callback(new Error(i18n.global.t('github.customURLHelper')));
        return;
    }
    callback();
};

const rules = reactive({
    name: [Rules.requiredInput],
    cloneURL: [{ validator: checkCloneURL, trigger: 'blur', required: true }],
    repo: [Rules.requiredInput],
    branch: [Rules.requiredInput, Rules.noSpace],
    projectPath: [{ validator: checkPath, trigger: 'blur', required: true }],
    keyMode: [Rules.requiredSelect],
    keyID: [{ validator: checkKey, trigger: 'change' }],
    domainMode: [Rules.requiredSelect],
    domainValue: [{ validator: checkDomain, trigger: 'change' }],
    customURL: [{ validator: checkCustomURL, trigger: 'blur' }],
    updateStrategy: [Rules.requiredSelect],
});

const loadOptions = async () => {
    const [keyRes, domainRes] = await Promise.all([listGithubKeyOptions(), listGithubDomains()]);
    keys.value = keyRes.data || [];
    domains.value = domainRes.data || [];
};

const acceptParams = (row?: Github.ProjectInfo) => {
    Object.assign(form, emptyForm());
    if (row) {
        Object.assign(form, {
            id: row.id,
            name: row.name,
            cloneURL: row.cloneURL,
            repo: row.repo,
            branch: row.branch,
            projectPath: row.projectPath,
            deployCommand: row.deployCommand,
            updateStrategy: row.updateStrategy,
            runAsUser: row.runAsUser,
            timeout: row.timeout,
            keyMode: row.keyMode,
            keyID: row.keyMode === 'custom' ? row.keyID : 0,
            domainMode: row.domainMode,
            websiteID: row.websiteID,
            domain: row.domain,
            customURL: row.customURL,
            enabled: row.enabled,
        });
    }
    drawerVisible.value = true;
    loadOptions().catch(() => {});
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
            const params = { ...form };
            if (params.keyMode !== 'custom') {
                params.keyID = 0;
            }
            if (params.id) {
                await updateGithubProject(params);
                MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
            } else {
                const res = await createGithubProject(params);
                MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
                emit('created', res.data);
            }
            drawerVisible.value = false;
            emit('search');
        } finally {
            loading.value = false;
        }
    });
};

defineExpose({ acceptParams });
</script>
