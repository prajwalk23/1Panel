<template>
    <div>
        <div class="flex flex-wrap items-center">
            <div class="flex flex-wrap items-center">
                <el-link v-if="isEE" underline="never" type="primary" @click="toEdition">
                    {{ $t('license.ee') }}
                </el-link>
                <el-link v-else-if="isMasterPro" underline="never" type="primary" @click="toLxware">
                    {{ $t('license.pro') }}
                </el-link>
                <el-link v-else-if="isOffline" underline="never" type="primary" @click="to1Panel">
                    {{ $t('license.offLine') }}
                </el-link>
                <el-link v-else underline="never" type="primary" @click="toEdition">
                    {{ $t('license.community') }}
                </el-link>
                <el-link underline="never" class="version" type="primary" @click="getVersionLog()">
                    {{ version }}
                </el-link>
                <el-badge
                    is-dot
                    v-if="isAdmin && !isOffline && !isEE"
                    class="-mt-0.5"
                    :hidden="version === 'Waiting' || !hasNewVersion"
                >
                    <el-link class="ml-2" underline="never" type="primary" @click="onLoadUpgradeInfo()">
                        {{ $t('commons.button.update') }}
                    </el-link>
                </el-badge>
                <el-tag v-if="version === 'Waiting'" round class="ml-2.5">{{ $t('setting.upgrading') }}</el-tag>
            </div>
        </div>

        <Upgrade ref="upgradeRef" @search="search" />
        <Releases ref="releasesRef" />

        <el-dialog
            v-model="noticeVisible"
            width="480px"
            align-center
            append-to-body
            class="upgrade-notice"
            :show-close="false"
            :close-on-click-modal="false"
        >
            <template #header>
                <div class="notice-banner">{{ $t('setting.versionUpdate') }}</div>
            </template>
            <h2 class="notice-title">{{ $t('setting.upgradeAvailable') }}</h2>
            <div class="notice-version">
                {{ $t('setting.upgradeLatest') }}:
                <span class="notice-version__value">{{ upgradeVersion }}</span>
            </div>
            <div class="notice-note">
                <MarkDownEditor v-if="upgradeInfo?.releaseNote" :content="upgradeInfo.releaseNote" />
            </div>
            <template #footer>
                <div class="notice-footer">
                    <el-button round @click="onIgnoreVersion">{{ $t('setting.upgradeIgnore') }}</el-button>
                    <el-button round type="primary" @click="onUpdateNow">{{ $t('setting.upgradeNow') }}</el-button>
                    <el-button round @click="noticeVisible = false">{{ $t('setting.upgradeDelay') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { getSettingBaseInfo, loadUpgradeInfo } from '@/api/modules/setting';
import Upgrade from '@/components/system-upgrade/upgrade/index.vue';
import Releases from '@/components/system-upgrade/releases/index.vue';
import MarkDownEditor from '@/components/mkdown-editor/index.vue';
import { IGNORED_KEY, PROMPTED_PREFIX } from '@/components/system-upgrade/status';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { onMounted, ref } from 'vue';
import { useGlobalStore } from '@/composables/useGlobalStore';

const { isOffline, isMasterPro, isEE, isIntl, isAdmin, hasNewVersion } = useGlobalStore();
const upgradeRef = ref();
const releasesRef = ref();

const version = ref<string>('');
const loading = ref(false);
const upgradeInfo = ref();
const upgradeVersion = ref();
const noticeVisible = ref(false);

const onIgnoreVersion = () => {
    try {
        localStorage.setItem(IGNORED_KEY, upgradeVersion.value);
    } catch (e) {
        // storage blocked: ignore only for this page load
    }
    noticeVisible.value = false;
};

const onUpdateNow = () => {
    noticeVisible.value = false;
    upgradeRef.value.acceptParams({ upgradeInfo: upgradeInfo.value, upgradeVersion: upgradeVersion.value });
};

const search = async () => {
    const res = await getSettingBaseInfo();
    version.value = res.data.forkVersion || res.data.systemVersion; // fork: show the full fork build
};

const getVersionLog = () => {
    if (isOffline.value) {
        return;
    }
    releasesRef.value.acceptParams();
};

const toLxware = () => {
    if (!isIntl.value) {
        window.open('https://www.lxware.cn/1panel' + '', '_blank', 'noopener,noreferrer');
    } else {
        window.open('https://1panel.pro/pricing' + '', '_blank', 'noopener,noreferrer');
    }
};

const to1Panel = () => {
    let url = isIntl.value ? 'https://1panel.pro' : 'https://1panel.cn';
    window.open(url, '_blank', 'noopener,noreferrer');
};

const toEdition = () => {
    if (!isIntl.value) {
        window.open('https://1panel.cn/versions.html' + '', '_blank', 'noopener,noreferrer');
    } else {
        window.open('https://1panel.pro/pricing' + '', '_blank', 'noopener,noreferrer');
    }
};

// auto: login-time check. Opens the popup only when an update exists, once per browser
// session per version (and never for an ignored version), and stays silent otherwise.
// Manual (Update button) is separate: it opens the upgrade drawer directly.
const onLoadUpgradeInfo = async (auto = false) => {
    loading.value = true;
    await loadUpgradeInfo()
        .then((res) => {
            loading.value = false;
            if (res.data.testVersion || res.data.newVersion || res.data.latestVersion) {
                const target = res.data.latestVersion || res.data.testVersion || res.data.newVersion;
                if (auto) {
                    const key = PROMPTED_PREFIX + target;
                    try {
                        if (localStorage.getItem(IGNORED_KEY) === target || sessionStorage.getItem(key)) {
                            return;
                        }
                        sessionStorage.setItem(key, '1');
                    } catch (e) {
                        // storage blocked: still prompt
                    }
                }
                upgradeInfo.value = res.data;
                upgradeVersion.value = target;
                if (auto) {
                    // login check: notice popup; "Update now" in it opens the upgrade drawer
                    noticeVisible.value = true;
                    return;
                }
                // footer Update button: upgrade drawer directly
                upgradeRef.value.acceptParams({ upgradeInfo: upgradeInfo.value, upgradeVersion: upgradeVersion.value });
            } else if (!auto) {
                MsgSuccess(i18n.global.t('setting.noUpgrade'));
                return;
            }
        })
        .catch(() => {
            loading.value = false;
        });
};

onMounted(async () => {
    await search();
    if (isAdmin.value && !isOffline.value && !isEE.value && version.value !== 'Waiting') {
        setTimeout(() => onLoadUpgradeInfo(true), 2000);
    }
});
</script>

<style lang="scss" scoped>
.line-height {
    line-height: 25px;
}
.notice-banner {
    margin: -16px -16px 0;
    padding: 16px 20px 48px;
    font-weight: 600;
    color: #fff;
    background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-light-3));
}
.notice-title {
    margin: 8px 0 16px;
    text-align: center;
    font-size: 22px;
    font-weight: 600;
}
.notice-version {
    margin-bottom: 12px;
    &__value {
        color: var(--el-color-primary);
        font-weight: 600;
    }
}
.notice-note {
    max-height: 240px;
    overflow-y: auto;
    padding: 8px 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    :deep(.md-editor-preview) {
        font-size: 14px;
    }
}
.notice-footer {
    display: flex;
    justify-content: center;
    gap: 12px;
}
:deep(.el-link__inner) {
    font-weight: 400;
}
.version {
    margin-left: 8px;
    font-size: 14px;
    color: var(--panel-color-primary-light-4);
    text-decoration: none;
    letter-spacing: 0.5px;
    cursor: pointer;
    font-family: auto;
}
</style>
