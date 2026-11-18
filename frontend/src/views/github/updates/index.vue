<template>
    <div v-loading="loading">
        <el-card>
            <template #header>
                <div class="flex items-center justify-between">
                    <span>{{ $t('github.updates') }}</span>
                    <el-button link type="primary" @click="load">{{ $t('commons.button.refresh') }}</el-button>
                </div>
            </template>
            <el-alert v-if="info && !info.isForkBuild" type="info" :closable="false" class="mb-3">
                <template #title>{{ $t('github.notForkBuild') }}</template>
            </el-alert>
            <el-alert v-if="info && info.systemStatus === 'Upgrading'" type="warning" :closable="false" class="mb-3">
                <template #title>{{ $t('github.upgrading') }}</template>
            </el-alert>
            <el-descriptions v-if="info" :column="1" border>
                <el-descriptions-item :label="$t('github.panelVersion')">
                    {{ info.panelVersion || '-' }}
                </el-descriptions-item>
                <el-descriptions-item :label="$t('github.forkVersion')">
                    {{ info.forkVersion || '-' }}
                </el-descriptions-item>
                <el-descriptions-item :label="$t('github.forkRepo')">
                    <el-link
                        type="primary"
                        :href="'https://github.com/' + info.forkRepo"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        {{ info.forkRepo }}
                    </el-link>
                </el-descriptions-item>
                <el-descriptions-item :label="$t('github.latestVersion')">
                    <span>{{ info.latestVersion || '-' }}</span>
                    <el-tag v-if="info.hasUpdate" type="danger" class="ml-2">{{ $t('github.newVersion') }}</el-tag>
                    <el-tag v-else-if="info.isForkBuild && info.latestVersion" type="success" class="ml-2">
                        {{ $t('github.upToDate') }}
                    </el-tag>
                </el-descriptions-item>
            </el-descriptions>

            <template v-if="info && info.hasUpdate">
                <el-alert v-if="info.baseChange" type="warning" :closable="false" class="mt-3">
                    <template #title>
                        {{ $t('github.baseChangeHelper', [info.panelVersion, info.latestBase]) }}
                    </template>
                </el-alert>
                <el-alert v-if="!info.assetReady" type="error" :closable="false" class="mt-3">
                    <template #title>{{ $t('github.assetMissing', [info.arch || '?']) }}</template>
                </el-alert>
                <div class="mt-3 flex gap-2">
                    <el-button
                        v-permission
                        type="primary"
                        :disabled="!info.assetReady || info.systemStatus === 'Upgrading'"
                        @click="onUpgrade"
                    >
                        {{ $t('github.update') }}
                    </el-button>
                    <el-button v-if="info.releaseURL" @click="openRelease">{{ $t('github.viewOnGithub') }}</el-button>
                </div>
            </template>
        </el-card>

        <el-card v-if="info && info.releaseNote" class="mt-3">
            <template #header>
                <span>{{ $t('github.releaseNotes') }}: {{ info.releaseName || info.latestVersion }}</span>
            </template>
            <MarkDownEditor :content="info.releaseNote" />
        </el-card>
    </div>
</template>

<script lang="ts" setup>
// fork: update the panel from the fork's GitHub Releases.
import { onMounted, ref } from 'vue';
import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import MarkDownEditor from '@/components/mkdown-editor/index.vue';
import { useGlobalStore } from '@/composables/useGlobalStore';
import { Github } from '@/api/interface/github';
import { forkUpgrade, loadForkInfo } from '@/api/modules/github';

const { isLoading, isOnRestart } = useGlobalStore();
const loading = ref(false);
const info = ref<Github.ForkInfo>();

const load = async () => {
    loading.value = true;
    try {
        const res = await loadForkInfo();
        info.value = res.data;
    } finally {
        loading.value = false;
    }
};

const openRelease = () => {
    if (info.value?.releaseURL) {
        window.open(info.value.releaseURL, '_blank', 'noopener,noreferrer');
    }
};

const onUpgrade = () => {
    if (!info.value) return;
    let message = i18n.global.t('github.updateHelper', [info.value.latestVersion]);
    if (info.value.baseChange) {
        message += ' ' + i18n.global.t('github.baseChangeHelper', [info.value.panelVersion, info.value.latestBase]);
    }
    ElMessageBox.confirm(message, i18n.global.t('github.update'), {
        confirmButtonText: i18n.global.t('commons.button.confirm'),
        cancelButtonText: i18n.global.t('commons.button.cancel'),
        type: 'warning',
    }).then(async () => {
        await forkUpgrade(info.value!.latestVersion);
        isLoading.value = true;
        isOnRestart.value = true;
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
    });
};

onMounted(() => {
    load().catch(() => {});
});
</script>
