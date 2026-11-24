<template>
    <DialogPro v-model="visible" :title="$t('github.deployNow')" size="small">
        <div v-loading="loading">
            <p class="mb-3">{{ $t('github.deployHelper', [projectName]) }}</p>
            <el-form label-position="top" @submit.prevent>
                <el-form-item :label="$t('github.deployScript')">
                    <el-select v-model="scriptID" class="w-full">
                        <el-option v-for="item in scripts" :key="item.id" :value="item.id" :label="item.name">
                            <span>{{ item.name }}</span>
                            <el-tag v-if="item.isDefault" size="small" class="ml-2">
                                {{ $t('github.defaultScript') }}
                            </el-tag>
                            <el-tag v-if="item.linked" size="small" type="success" class="ml-1">
                                {{ $t('github.linkedScript') }}
                            </el-tag>
                        </el-option>
                    </el-select>
                </el-form-item>
            </el-form>
        </div>
        <template #footer>
            <el-button @click="visible = false">{{ $t('commons.button.cancel') }}</el-button>
            <el-button type="primary" :disabled="!scriptID || loading" @click="onRun">
                {{ $t('commons.button.confirm') }}
            </el-button>
        </template>
    </DialogPro>
</template>

<script lang="ts" setup>
// fork: "Pull now" with the choice of which script to run afterwards.
import { ref } from 'vue';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import { deployGithubProject, listGithubScripts } from '@/api/modules/github';

const emit = defineEmits<{ (e: 'search'): void }>();

const visible = ref(false);
const loading = ref(false);
const projectID = ref(0);
const projectName = ref('');
const scriptID = ref(0);
const scripts = ref<Github.ScriptInfo[]>([]);

const acceptParams = async (id: number, name: string) => {
    projectID.value = id;
    projectName.value = name;
    scriptID.value = 0;
    scripts.value = [];
    visible.value = true;
    loading.value = true;
    try {
        const res = await listGithubScripts(id);
        scripts.value = res.data || [];
        scriptID.value = scripts.value.find((item) => item.linked)?.id || scripts.value[0]?.id || 0;
    } finally {
        loading.value = false;
    }
};

const onRun = async () => {
    loading.value = true;
    try {
        await deployGithubProject({ id: projectID.value, scriptID: scriptID.value });
        MsgSuccess(i18n.global.t('github.deployStarted'));
        visible.value = false;
        emit('search');
    } finally {
        loading.value = false;
    }
};

defineExpose({ acceptParams });
</script>
