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
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
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
    resetDotOnUpdateDone(version.value);
};

// update complete = the running version differs from the one this browser last saw (survives the
// post-update reload/re-login): drop the old red dot and make the check due, so the dot only comes
// back from a fresh check against the new version, never from the previous result.
const resetDotOnUpdateDone = (current: string) => {
    if (!current || current === 'Waiting') {
        return;
    }
    const key = 'panelVersionSeen';
    try {
        const seen = localStorage.getItem(key);
        if (seen && seen !== current) {
            hasNewVersion.value = false;
            sessionStorage.removeItem('upgradePrompted');
            localStorage.removeItem(CHECK_KEY);
        }
        localStorage.setItem(key, current);
    } catch (e) {
        // storage may be blocked; the server re-check still runs
    }
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

// The server never caches the check. This browser asks on login (login and logout clear the key),
// then at most once every 24 hours, and whenever Update is clicked.
const CHECK_KEY = 'upgradeChecked';
const CHECK_INTERVAL = 24 * 60 * 60 * 1000;

const isCheckDue = (): boolean => {
    try {
        const last = Number(localStorage.getItem(CHECK_KEY));
        return !last || Date.now() - last >= CHECK_INTERVAL;
    } catch (e) {
        return true;
    }
};

const markChecked = () => {
    try {
        localStorage.setItem(CHECK_KEY, String(Date.now()));
    } catch (e) {
        // storage may be blocked; the next load just checks again
    }
};

// returns true when a new version exists; fills upgradeInfo and upgradeVersion
const fetchUpgrade = async (): Promise<boolean> => {
    markChecked();
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
    planRef.value.acceptParams({
        version: newVersion || '',
        plan: plan.value,
        releaseNote: upgradeInfo.value?.releaseNote || '',
    });
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
    if (!isCheckDue()) {
        return;
    }
    try {
        // the check also clears a stale persisted red dot, even when a plan or ignore is set
        if (!(await fetchUpgrade())) {
            return;
        }
        if (plan.value.mode !== 'none') {
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

// silent re-check for the daily timer, and after an upgrade finishes or a scheduled upgrade ran
const refreshUpgradeStatus = async () => {
    if (!canPlan.value) {
        return;
    }
    try {
        await fetchUpgrade();
    } catch (e) {
        // request errors are already reported by the http layer
    }
};

// version changed (upgrade done): drop the dot at once, then confirm with the server
watch(version, (val, old) => {
    if (!old || val === old || val === 'Waiting') {
        return;
    }
    hasNewVersion.value = false;
    refreshUpgradeStatus();
});

let statusTimer: ReturnType<typeof setInterval> | undefined;

onMounted(async () => {
    await search();
    promptNewVersion();
    statusTimer = setInterval(refreshUpgradeStatus, CHECK_INTERVAL);
});

onBeforeUnmount(() => {
    if (statusTimer) {
        clearInterval(statusTimer);
    }
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
