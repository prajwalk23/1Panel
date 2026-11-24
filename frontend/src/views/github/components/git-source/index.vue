<template>
    <div class="git-source">
        <el-divider v-if="withSwitch" content-position="left">
            <el-text type="info" size="small">{{ $t('github.menu') }}</el-text>
        </el-divider>
        <el-form-item v-if="withSwitch" :label="$t('github.sourceSwitch')">
            <el-switch v-model="switchValue" />
            <span class="input-help">{{ $t('github.sourceSwitchHelper') }}</span>
        </el-form-item>
        <template v-if="!withSwitch || switchValue">
            <SourceFields :source="source" :repo="repo" :errors="errors" />
            <el-form-item v-if="showPath" :label="$t('github.pullDirectory')" :error="errors.path">
                <el-input v-model.trim="pathValue">
                    <template #append>
                        <el-button icon="Folder" @click="fileRef.acceptParams({ dir: true })" />
                    </template>
                </el-input>
                <span class="input-help">{{ $t('github.projectPathHelper') }}</span>
            </el-form-item>
        </template>
    </div>
    <FileList ref="fileRef" @choose="(dir: string) => (pathValue = dir)" />
</template>

<script lang="ts" setup>
// fork: the one create form of a Git source (GitHub tab, website create, runtime create).
// Create only: it asks for the repository, how to sign in and the branch. Everything else
// (scripts, webhook, deploy records, ignored paths) is set in the Git Manager afterwards.
import { computed, onMounted, reactive, ref } from 'vue';
import { Github } from '@/api/interface/github';
import FileList from '@/components/file-list/index.vue';
import i18n from '@/lang';
import SourceFields from '@/views/github/components/source-fields/index.vue';
import { emptySource, sourceError, useGitRepo } from '@/views/github/composables/use-git-repo';

const props = defineProps({
    source: {
        type: Object as () => Github.Source,
        required: true,
    },
    // Show the "Source from GitHub" switch (website and runtime create). The form shows when it is on.
    withSwitch: {
        type: Boolean,
        default: false,
    },
    enabled: {
        type: Boolean,
        default: true,
    },
    // Ask for the folder the code is pulled into (GitHub tab).
    showPath: {
        type: Boolean,
        default: false,
    },
    path: {
        type: String,
        default: '',
    },
});
const emit = defineEmits<{
    (e: 'update:enabled', value: boolean): void;
    (e: 'update:path', value: string): void;
}>();

const fileRef = ref();
const errors = reactive({ url: '', token: '', key: '', branch: '', path: '' });
const repo = useGitRepo(props.source);

const switchValue = computed({
    get: () => props.enabled,
    set: (value: boolean) => {
        emit('update:enabled', value);
        if (value) {
            repo.loadKeys().catch(() => {});
        }
    },
});
const pathValue = computed({
    get: () => props.path,
    set: (value: string) => emit('update:path', value),
});

const clearErrors = () => {
    errors.url = errors.token = errors.key = errors.branch = errors.path = '';
};

// Checks the form and shows what is wrong under the field. Returns true when it can be submitted.
const validate = (): boolean => {
    clearErrors();
    if (props.withSwitch && !props.enabled) {
        return true;
    }
    const source = props.source;
    const message = sourceError(source);
    if (message) {
        if (!source.branch.trim() && message === i18n.global.t('github.selectBranch')) errors.branch = message;
        else if (source.authMode === 'token' && !source.token.trim()) errors.token = message;
        else errors.url = message;
        return false;
    }
    if (props.showPath && (!props.path || !props.path.startsWith('/') || props.path === '/')) {
        errors.path = i18n.global.t('github.projectPathHelper');
        return false;
    }
    return true;
};

const reset = () => {
    Object.assign(props.source, emptySource());
    clearErrors();
    repo.onUrlChange();
};

onMounted(() => {
    if (!props.withSwitch || props.enabled) {
        repo.loadKeys().catch(() => {});
    }
});

defineExpose({ validate, reset });
</script>
