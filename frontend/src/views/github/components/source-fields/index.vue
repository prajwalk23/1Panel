<template>
    <el-form-item :label="$t('github.repoURL')" :error="errors.url">
        <el-input
            v-model.trim="source.cloneURL"
            :placeholder="$t('github.repoURLPlaceholder')"
            @change="repo.onUrlChange()"
        />
        <span class="input-help">{{ $t('github.repoURLExample') }}</span>
    </el-form-item>

    <el-form-item :label="$t('github.authentication')">
        <el-radio-group v-model="source.authMode" @change="repo.onAuthChange()">
            <el-radio-button value="public">{{ $t('github.authPublic') }}</el-radio-button>
            <el-radio-button value="token">{{ $t('github.authToken') }}</el-radio-button>
            <el-radio-button value="ssh">{{ $t('github.authSSH') }}</el-radio-button>
        </el-radio-group>
    </el-form-item>

    <template v-if="source.authMode === 'token'">
        <el-form-item :label="$t('github.tokenUser')">
            <el-input
                v-model.trim="source.tokenUser"
                :placeholder="$t('github.tokenUserPlaceholder')"
                @change="repo.resetCheck()"
            />
        </el-form-item>
        <el-form-item :label="$t('github.tokenValue')" :error="errors.token">
            <el-input
                v-model.trim="source.token"
                type="password"
                show-password
                autocomplete="new-password"
                :placeholder="tokenStored ? $t('github.tokenKept') : ''"
                @change="repo.resetCheck()"
            />
            <span class="input-help">{{ $t('github.tokenHelper') }}</span>
        </el-form-item>
    </template>

    <el-form-item v-if="source.authMode === 'ssh'" :label="$t('github.sshKey')" :error="errors.key">
        <div class="flex w-full gap-2">
            <el-select v-model="source.keyID" class="flex-1" @change="repo.resetCheck()">
                <el-option v-if="repo.keys.value.length === 0" :value="0" :label="$t('github.sshKeyShared')" />
                <el-option v-for="item in repo.keys.value" :key="item.id" :value="item.id" :label="item.name" />
            </el-select>
            <el-button :loading="repo.creatingKey.value" @click="repo.generateKey()">
                {{ $t('github.generateProjectKey') }}
            </el-button>
        </div>
        <template v-if="repo.selectedKey.value">
            <div class="mt-2 flex w-full items-start gap-1">
                <el-input
                    :model-value="repo.selectedKey.value.publicKey"
                    type="textarea"
                    :rows="3"
                    readonly
                    class="font-mono"
                />
                <CopyButton :content="repo.selectedKey.value.publicKey" />
            </div>
            <el-link
                v-if="repo.githubRepoName.value"
                type="primary"
                :underline="false"
                :href="'https://github.com/' + repo.githubRepoName.value + '/settings/keys/new'"
                target="_blank"
                rel="noopener noreferrer"
                class="mt-1"
            >
                {{ $t('github.openDeployKeys') }}
            </el-link>
        </template>
        <span class="input-help">{{ $t('github.sshKeyHelper') }}</span>
    </el-form-item>

    <el-form-item :label="$t('github.branchOrTag')" :error="errors.branch">
        <div class="flex w-full gap-2">
            <el-select
                v-model="source.branch"
                filterable
                allow-create
                default-first-option
                class="flex-1"
                :placeholder="$t('github.selectBranch')"
            >
                <el-option v-for="item in repo.branchOptions.value" :key="item" :value="item" :label="item" />
            </el-select>
            <el-button v-if="testButton" :loading="repo.checking.value" @click="repo.check()">
                {{ $t('github.testConnection') }}
            </el-button>
            <el-tooltip v-else :content="$t('github.refreshBranches')" placement="top">
                <el-button :loading="repo.checking.value" icon="Refresh" @click="repo.check()" />
            </el-tooltip>
        </div>
        <span v-if="testButton" class="input-help">{{ $t('github.testConnectionHelper') }}</span>
        <el-alert
            v-if="repo.checkResult.value"
            :type="repo.checkResult.value.ok ? 'success' : 'error'"
            :closable="false"
            class="mt-2"
        >
            <template #title>
                <span v-if="repo.checkResult.value.ok">
                    {{ $t('github.repoOK', [repo.checkResult.value.branches.length]) }}
                </span>
                <template v-else>
                    <div>{{ $t('github.repoFail') }}</div>
                    <div class="text-xs break-all">{{ repo.checkResult.value.message }}</div>
                    <div v-if="source.authMode === 'ssh'" class="text-xs">{{ $t('github.repoNeedsKey') }}</div>
                </template>
            </template>
        </el-alert>
    </el-form-item>
</template>

<script lang="ts" setup>
// fork: repository URL, authentication and branch. The block the create form, the GitHub project
// editor and the Git Manager repository settings have in common.
import { Github } from '@/api/interface/github';
import CopyButton from '@/components/copy-button/index.vue';
import { useGitRepo } from '@/views/github/composables/use-git-repo';

defineProps({
    source: {
        type: Object as () => Github.Source,
        required: true,
    },
    repo: {
        type: Object as () => ReturnType<typeof useGitRepo>,
        required: true,
    },
    errors: {
        type: Object as () => { url: string; token: string; key: string; branch: string },
        default: () => ({ url: '', token: '', key: '', branch: '' }),
    },
    // A token is stored already: the token field may stay empty.
    tokenStored: {
        type: Boolean,
        default: false,
    },
    // Edit forms show a "Test connection" button, the create form a refresh icon.
    testButton: {
        type: Boolean,
        default: false,
    },
});
</script>
