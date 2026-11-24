<template>
    <DialogPro v-model="visible" :title="$t('github.sourceTitle')" size="large" @close="onClose">
        <el-form ref="formRef" label-position="top" :model="form" :rules="rules" v-loading="loading" @submit.prevent>
            <el-form-item :label="$t('github.name')" prop="name">
                <el-input v-model.trim="form.name" :placeholder="repoName" />
            </el-form-item>
            <GitSource ref="sourceRef" :source="source" show-path v-model:path="form.projectPath" />
        </el-form>
        <template #footer>
            <el-button @click="visible = false">{{ $t('commons.button.cancel') }}</el-button>
            <el-button v-permission v-node-admin type="primary" :disabled="loading" @click="onSubmit">
                {{ $t('github.pullProject') }}
            </el-button>
        </template>
    </DialogPro>
</template>

<script lang="ts" setup>
// fork: create a GitHub project. Short on purpose: the repository, how to sign in, the branch and
// the folder. Scripts, webhook, deploy records and ignored paths are set when editing.
import { computed, reactive, ref } from 'vue';
import { ElForm } from 'element-plus';
import i18n from '@/lang';
import { Github } from '@/api/interface/github';
import { createGithubProject, deployGithubProject } from '@/api/modules/github';
import { MsgSuccess } from '@/utils/message';
import GitSource from '@/views/github/components/git-source/index.vue';
import { buildCloneURL, emptySource, keyModeOf, repoFromInput } from '@/views/github/composables/use-git-repo';

type FormInstance = InstanceType<typeof ElForm>;

const emit = defineEmits<{ (e: 'search'): void }>();

const visible = ref(false);
const loading = ref(false);
const formRef = ref<FormInstance>();
const sourceRef = ref<InstanceType<typeof GitSource>>();
const source = reactive<Github.Source>(emptySource());
const form = reactive({ name: '', projectPath: '' });

const repoName = computed(() => repoFromInput(source.cloneURL).split('/')[1] || '');
const rules = reactive({
    name: [
        {
            validator: (_rule: any, value: string, callback: any) => {
                if (!value && !repoName.value) {
                    callback(new Error(i18n.global.t('commons.rule.requiredInput')));
                    return;
                }
                callback();
            },
            trigger: 'blur',
        },
    ],
});

const acceptParams = () => {
    Object.assign(source, emptySource());
    form.name = '';
    form.projectPath = '';
    visible.value = true;
};

const onClose = () => {
    formRef.value?.resetFields();
};

const onSubmit = async () => {
    const formOK = await formRef.value?.validate().catch(() => false);
    if (!formOK || !sourceRef.value?.validate()) return;
    loading.value = true;
    try {
        const cloneURL = buildCloneURL(source);
        const params: Github.ProjectOperate = {
            name: form.name || repoName.value,
            cloneURL: cloneURL,
            repo: repoFromInput(cloneURL),
            branch: source.branch,
            projectPath: form.projectPath,
            updateStrategy: 'ff-only',
            timeout: 600,
            keyMode: keyModeOf(source),
            keyID: source.authMode === 'ssh' ? source.keyID : 0,
            tokenUser: source.authMode === 'token' ? source.tokenUser : '',
            token: source.authMode === 'token' ? source.token : '',
            keepRecords: 5,
            ignorePaths: '',
            enabled: true,
        };
        const res = await createGithubProject(params);
        await deployGithubProject({ id: res.data.id });
        MsgSuccess(i18n.global.t('github.pullDone'));
        visible.value = false;
        emit('search');
    } finally {
        loading.value = false;
    }
};

defineExpose({ acceptParams });
</script>
