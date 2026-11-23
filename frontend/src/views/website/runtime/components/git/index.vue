<template>
    <div class="runtime-git">
        <el-form-item :label="$t('runtime.gitSource')">
            <el-switch v-model="enabled" />
            <span class="input-help">{{ $t('runtime.gitSourceHelper') }}</span>
        </el-form-item>
        <template v-if="enabled">
            <el-form-item :label="$t('github.repo')" prop="gitCloneURL" :rules="urlRules">
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
                        </template>
                    </template>
                </el-alert>
            </el-form-item>

            <el-form-item :label="$t('github.branch')" prop="gitBranch">
                <div class="flex w-full gap-2">
                    <el-select
                        v-model="runtime.gitBranch"
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

            <el-form-item :label="$t('runtime.gitKey')" prop="gitKeyID">
                <div class="flex w-full gap-2">
                    <el-select v-model="runtime.gitKeyID" class="flex-1" @change="onKeyChange">
                        <el-option :label="$t('runtime.gitKeyShared')" :value="0" />
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
                <span v-else class="input-help">{{ $t('runtime.gitKeyHelper') }}</span>
            </el-form-item>
        </template>
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue';
import { useVModel } from '@vueuse/core';
import { Github } from '@/api/interface/github';
import { checkGithubRepo, createGithubKey, listGithubKeyOptions } from '@/api/modules/github';
import CopyButton from '@/components/copy-button/index.vue';
import i18n from '@/lang';
import { MsgError, MsgSuccess } from '@/utils/message';

const props = defineProps({
    mode: {
        type: String,
        required: true,
    },
    modelValue: {
        type: Object,
        required: true,
    },
});
const emit = defineEmits(['update:modelValue']);
const runtime = useVModel(props, 'modelValue', emit);

const keys = ref<Github.KeyOption[]>([]);
const branches = ref<string[]>([]);
const checkResult = ref<Github.RepoCheckInfo>();
const repoInput = ref('');
const checking = ref(false);
const creatingKey = ref(false);
const showKeyCreate = ref(false);
const newKeyName = ref('');

const enabled = computed({
    get: () => runtime.value.gitMode === 'git',
    set: (val: boolean) => {
        runtime.value.gitMode = val ? 'git' : 'none';
        if (val) {
            if (!runtime.value.gitBranch) {
                runtime.value.gitBranch = 'main';
            }
            // Update strategy is fixed to fast-forward; kept for the backend.
            runtime.value.gitUpdateStrategy = 'ff-only';
            if (runtime.value.gitKeyID === undefined) {
                runtime.value.gitKeyID = 0;
            }
        }
    },
});

const selectedKey = computed(() => keys.value.find((item) => item.id === runtime.value.gitKeyID));
const branchOptions = computed(() => {
    const list = [...branches.value];
    if (runtime.value.gitBranch && !list.includes(runtime.value.gitBranch)) {
        list.unshift(runtime.value.gitBranch);
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
    return runtime.value.gitKeyID ? `git@github.com:${name}.git` : `https://github.com/${name}.git`;
};

const syncCloneURL = () => {
    const url = buildCloneURL();
    if (url) {
        runtime.value.gitCloneURL = url;
    }
};

const onRepoChange = () => {
    checkResult.value = undefined;
    branches.value = [];
    syncCloneURL();
};

const onKeyChange = () => {
    checkResult.value = undefined;
    syncCloneURL();
};

const onCheck = async () => {
    const cloneURL = buildCloneURL();
    if (!cloneURL) {
        MsgError(i18n.global.t('github.repoHelper'));
        return;
    }
    runtime.value.gitCloneURL = cloneURL;
    checking.value = true;
    try {
        const res = await checkGithubRepo({
            cloneURL: cloneURL,
            keyMode: runtime.value.gitKeyID ? 'custom' : 'none',
            keyID: runtime.value.gitKeyID || 0,
        });
        checkResult.value = res.data;
        branches.value = res.data.branches || [];
        if (res.data.ok && (!runtime.value.gitBranch || !branches.value.includes(runtime.value.gitBranch))) {
            runtime.value.gitBranch = res.data.defaultBranch || branches.value[0] || 'main';
        }
    } finally {
        checking.value = false;
    }
};

const loadKeys = async () => {
    const res = await listGithubKeyOptions();
    keys.value = res.data || [];
};
const onCreateKey = async () => {
    creatingKey.value = true;
    try {
        const res = await createGithubKey(newKeyName.value, 'shared');
        await loadKeys();
        runtime.value.gitKeyID = res.data.id;
        newKeyName.value = '';
        showKeyCreate.value = false;
        checkResult.value = undefined;
        syncCloneURL();
        MsgSuccess(i18n.global.t('github.keyCreated'));
    } finally {
        creatingKey.value = false;
    }
};

const urlRules = computed(() => {
    if (!enabled.value) {
        return [];
    }
    return [
        {
            required: true,
            validator: (_rule: any, _value: string, callback: (e?: Error) => void) => {
                if (!buildCloneURL()) {
                    callback(new Error(i18n.global.t('runtime.gitCloneURLRequired')));
                    return;
                }
                callback();
            },
            trigger: 'blur',
        },
    ];
});

watch(repoInput, () => syncCloneURL());

onMounted(() => {
    if (!runtime.value.gitMode) {
        runtime.value.gitMode = 'none';
    }
    if (runtime.value.gitCloneURL) {
        repoInput.value = runtime.value.gitCloneURL;
    }
    loadKeys().catch(() => {
        keys.value = [];
    });
});
</script>
