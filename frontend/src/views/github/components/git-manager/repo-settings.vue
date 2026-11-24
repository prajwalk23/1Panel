<template>
    <DialogPro v-model="visible" :title="$t('github.repoSettings')" size="large">
        <el-form label-position="top" v-loading="loading" @submit.prevent>
            <SourceFields :source="source" :repo="repo" :errors="errors" :token-stored="tokenStored" test-button />
            <el-row :gutter="20">
                <el-col :span="12">
                    <el-form-item :label="$t('github.timeout')">
                        <el-input-number v-model="form.timeout" :min="30" :max="86400" :step="60" />
                    </el-form-item>
                </el-col>
                <el-col :span="12">
                    <el-form-item :label="$t('github.deploymentRecords')">
                        <el-select v-model="form.keepRecords" class="w-full">
                            <el-option
                                v-for="item in [5, 10, 20, 30]"
                                :key="item"
                                :value="item"
                                :label="String(item)"
                            />
                        </el-select>
                        <span class="input-help">{{ $t('github.deploymentRecordsHelper') }}</span>
                    </el-form-item>
                </el-col>
            </el-row>
            <el-form-item>
                <template #label>
                    <span>{{ $t('github.ignorePaths') }}</span>
                    <span class="ml-2 text-xs text-gray-400">{{ $t('github.optional') }}</span>
                </template>
                <el-input
                    v-model="form.ignorePaths"
                    type="textarea"
                    :rows="4"
                    class="font-mono"
                    placeholder="/.env&#10;/storage&#10;/bootstrap/cache"
                />
                <span class="input-help">{{ $t('github.ignorePathsHelper') }}</span>
            </el-form-item>
            <el-alert type="info" :closable="false" :title="$t('github.savingNoDeploy')" />
        </el-form>
        <template #footer>
            <el-button @click="visible = false">{{ $t('commons.button.cancel') }}</el-button>
            <el-button v-permission v-node-admin type="primary" :disabled="loading" @click="onSave">
                {{ $t('commons.button.confirm') }}
            </el-button>
        </template>
    </DialogPro>
</template>

<script lang="ts" setup>
// fork: repository settings of the Git Manager. Saving does not deploy anything.
import { reactive, ref } from 'vue';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import { updateGithubProject } from '@/api/modules/github';
import SourceFields from '@/views/github/components/source-fields/index.vue';
import {
    buildCloneURL,
    emptySource,
    keyModeOf,
    repoFromInput,
    sourceError,
    useGitRepo,
} from '@/views/github/composables/use-git-repo';

const emit = defineEmits<{ (e: 'saved'): void }>();

const visible = ref(false);
const loading = ref(false);
const tokenStored = ref(false);
const projectID = ref<number | undefined>();
const project = ref<Github.ProjectInfo>();
const errors = reactive({ url: '', token: '', key: '', branch: '' });
const form = reactive({ timeout: 600, keepRecords: 5, ignorePaths: '' });
const source = reactive<Github.Source>(emptySource());
const repo = useGitRepo(source, { keepBranch: true, projectID: projectID, tokenStored: tokenStored });

const acceptParams = (row: Github.ProjectInfo) => {
    project.value = row;
    projectID.value = row.id;
    tokenStored.value = row.keyMode === 'token' && row.hasToken;
    Object.assign(form, {
        timeout: row.timeout,
        keepRecords: row.keepRecords || 5,
        ignorePaths: row.ignorePaths || '',
    });
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
    visible.value = true;
    repo.loadKeys().catch(() => {});
};

const onSave = async () => {
    errors.url = errors.token = errors.key = errors.branch = '';
    const message = sourceError(source, tokenStored.value);
    if (message) {
        if (!source.branch.trim()) errors.branch = message;
        else if (source.authMode === 'token' && !source.token.trim() && !tokenStored.value) errors.token = message;
        else errors.url = message;
        return;
    }
    const row = project.value as Github.ProjectInfo;
    loading.value = true;
    try {
        const cloneURL = buildCloneURL(source);
        await updateGithubProject({
            id: row.id,
            name: row.name,
            cloneURL: cloneURL,
            repo: repoFromInput(cloneURL),
            branch: source.branch,
            projectPath: row.projectPath,
            updateStrategy: row.updateStrategy,
            timeout: form.timeout,
            keyMode: keyModeOf(source),
            keyID: source.authMode === 'ssh' ? source.keyID : 0,
            tokenUser: source.authMode === 'token' ? source.tokenUser : '',
            token: source.authMode === 'token' ? source.token : '',
            keepRecords: form.keepRecords,
            ignorePaths: form.ignorePaths,
            enabled: row.enabled,
        });
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        visible.value = false;
        emit('saved');
    } finally {
        loading.value = false;
    }
};

defineExpose({ acceptParams });
</script>
