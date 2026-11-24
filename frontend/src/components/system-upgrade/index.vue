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
                    <el-link class="ml-2" underline="never" type="primary" @click="onLoadUpgradeInfo">
                        {{ $t('commons.button.update') }}
                    </el-link>
                </el-badge>
                <el-tag v-if="version === 'Waiting'" round class="ml-2.5">{{ $t('setting.upgrading') }}</el-tag>
                <el-tag
                    v-if="showPlanTag"
                    round
                    type="info"
                    class="ml-2.5 cursor-pointer"
                    @click="openPlan(upgradeVersion)"
                >
                    {{ planTagText }}
                </el-tag>
            </div>
        </div>

        <Upgrade ref="upgradeRef" @search="search" />
        <Releases ref="releasesRef" />
        <Plan ref="planRef" @upgrade="openUpgrade" @changed="loadPlan" />
    </div>
</template>

<script setup lang="ts">
import { getSettingBaseInfo, loadUpgradeInfo, loadUpgradePlan } from '@/api/modules/setting';
import { Setting } from '@/api/interface/setting';
import Upgrade from '@/components/system-upgrade/upgrade/index.vue';
import Releases from '@/components/system-upgrade/releases/index.vue';
import Plan from '@/components/system-upgrade/plan/index.vue';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';
import { computed, onMounted, ref } from 'vue';
import { useGlobalStore } from '@/composables/useGlobalStore';

const { isOffline, isMasterPro, isEE, isIntl, isAdmin, hasNewVersion } = useGlobalStore();
const upgradeRef = ref();
const releasesRef = ref();
const planRef = ref();
const plan = ref<Setting.UpgradePlan>({ mode: 'none', time: '' });

const canPlan = computed(() => isAdmin.value && !isOffline.value && !isEE.value);
const showPlanTag = computed(() => canPlan.value && plan.value.mode !== 'none');
const planTagText = computed(() => {
    switch (plan.value.mode) {
        case 'once':
            return i18n.global.t('setting.upgradePlanOnce', [plan.value.time]);
        case 'daily':
            return i18n.global.t('setting.upgradePlanDaily', [plan.value.time]);
        default:
            return i18n.global.t('setting.upgradePlanIgnore');
    }
});

const version = ref<string>('');
const loading = ref(false);
const upgradeInfo = ref();
const upgradeVersion = ref();

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

// returns true when a new version exists; fills upgradeInfo and upgradeVersion
const fetchUpgrade = async (): Promise<boolean> => {
    const res = await loadUpgradeInfo();
    if (!(res.data.testVersion || res.data.newVersion || res.data.latestVersion)) {
        hasNewVersion.value = false;
        return false;
    }
    hasNewVersion.value = true;
    upgradeInfo.value = res.data;
    if (upgradeInfo.value.latestVersion) {
        upgradeVersion.value = upgradeInfo.value.latestVersion;
    } else if (upgradeInfo.value.testVersion) {
        upgradeVersion.value = upgradeInfo.value.testVersion;
    } else if (upgradeInfo.value.newVersion) {
        upgradeVersion.value = upgradeInfo.value.newVersion;
    }
    return true;
};

const openUpgrade = () => {
    upgradeRef.value.acceptParams({ upgradeInfo: upgradeInfo.value, upgradeVersion: upgradeVersion.value });
};

const openPlan = (newVersion?: string) => {
    planRef.value.acceptParams({ version: newVersion || '', plan: plan.value });
};

const loadPlan = async () => {
    if (!canPlan.value) {
        return;
    }
    try {
        const res = await loadUpgradePlan();
        plan.value = res.data || { mode: 'none', time: '' };
    } catch (e) {
        plan.value = { mode: 'none', time: '' };
    }
};

const onLoadUpgradeInfo = async () => {
    loading.value = true;
    try {
        if (await fetchUpgrade()) {
            openUpgrade();
        } else {
            MsgSuccess(i18n.global.t('setting.noUpgrade'));
        }
    } catch (e) {
        // request errors are already reported by the http layer
    } finally {
        loading.value = false;
    }
};

// popup once per browser session and version, only when no plan is set yet
const promptNewVersion = async () => {
    if (!canPlan.value) {
        return;
    }
    await loadPlan();
    if (plan.value.mode !== 'none') {
        return;
    }
    try {
        if (!(await fetchUpgrade())) {
            return;
        }
        const key = 'upgradePrompted';
        if (sessionStorage.getItem(key) === upgradeVersion.value) {
            return;
        }
        sessionStorage.setItem(key, upgradeVersion.value);
        openPlan(upgradeVersion.value);
    } catch (e) {
        // sessionStorage may be blocked, or the check failed; skip the popup
    }
};

onMounted(async () => {
    await search();
    promptNewVersion();
});
</script>

<style lang="scss" scoped>
.line-height {
    line-height: 25px;
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
