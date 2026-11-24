<template>
    <div v-loading="loading" class="git-manager">
        <el-alert v-if="failed" type="warning" :closable="false" :title="$t('github.linkedProjectMissing')" />
        <el-tabs v-else-if="project" v-model="tab">
            <el-tab-pane :label="$t('github.tabRepository')" name="repository">
                <RepositoryTab v-if="tab === 'repository'" :project-id="project.id" @goto="tab = $event" />
            </el-tab-pane>
            <el-tab-pane :label="$t('github.tabDeployments')" name="deployments">
                <DeploymentsTab v-if="tab === 'deployments'" :project-id="project.id" />
            </el-tab-pane>
            <el-tab-pane :label="$t('github.tabScript')" name="script">
                <ScriptsTab v-if="tab === 'script'" :project-id="project.id" :project-name="project.name" />
            </el-tab-pane>
            <el-tab-pane :label="$t('github.tabWebhook')" name="webhook">
                <WebhookTab v-if="tab === 'webhook'" :project-id="project.id" @goto="tab = $event" />
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
// fork: the Git Manager of a website or a runtime that takes its code from a repository.
// Edit side of the Git source: repository settings, deploy history with rollback, scripts, webhook.
import { onMounted, ref } from 'vue';
import { Github } from '@/api/interface/github';
import { loadLinkedGithubProject } from '@/api/modules/github';
import RepositoryTab from '@/views/github/components/git-manager/repository.vue';
import DeploymentsTab from '@/views/github/components/git-manager/deployments.vue';
import ScriptsTab from '@/views/github/components/git-manager/script-list.vue';
import WebhookTab from '@/views/github/components/git-manager/webhook.vue';

const props = defineProps({
    websiteId: {
        type: Number,
        default: 0,
    },
    runtimeId: {
        type: Number,
        default: 0,
    },
});

const loading = ref(false);
const failed = ref(false);
const tab = ref('repository');
const project = ref<Github.ProjectInfo>();

const load = async () => {
    loading.value = true;
    failed.value = false;
    project.value = undefined;
    tab.value = 'repository';
    try {
        const res = await loadLinkedGithubProject({
            websiteID: props.websiteId || undefined,
            runtimeID: props.runtimeId || undefined,
        });
        project.value = res.data;
    } catch {
        failed.value = true;
    } finally {
        loading.value = false;
    }
};

onMounted(() => {
    load();
});

defineExpose({ load });
</script>
