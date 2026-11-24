<template>
    <el-dialog
        v-model="open"
        :title="$t('website.newSSL')"
        width="520px"
        append-to-body
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        :show-close="!running"
        :before-close="handleClose"
    >
        <div class="flex flex-col gap-3">
            <div class="flex items-center gap-2">
                <el-icon v-if="running" class="is-loading"><Loading /></el-icon>
                <el-icon v-else-if="state === 'success'" color="var(--el-color-success)"><CircleCheckFilled /></el-icon>
                <el-icon v-else color="var(--el-color-danger)"><CircleCloseFilled /></el-icon>
                <span>{{ statusText }}</span>
            </div>
            <el-text type="info" size="small" v-if="domainText">{{ domainText }}</el-text>
            <el-text type="info" size="small" v-if="running || state === 'failed'">
                {{ $t('website.newSSLAttempt', { current: attempt, max: MAX_ATTEMPTS }) }}
            </el-text>
            <el-alert v-if="state === 'failed' && errorMsg" type="error" :closable="false" show-icon>
                <span class="whitespace-pre-line break-all">{{ errorMsg }}</span>
            </el-alert>
        </div>
        <template #footer>
            <el-button v-if="hasLog" @click="openLog">{{ $t('commons.button.log') }}</el-button>
            <el-button v-if="state === 'failed'" type="primary" @click="start(true)">
                {{ $t('commons.button.retry') }}
            </el-button>
            <el-button :disabled="running" @click="handleClose()">
                {{ $t('commons.button.close') }}
            </el-button>
        </template>
        <Log ref="logRef" :heightDiff="220" />
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, ref } from 'vue';
import i18n from '@/lang';
import Log from '@/components/log/file-drawer/index.vue';
import { createSSL, deleteSSL, getSSL, listDomains, obtainSSL } from '@/api/modules/website';

const MAX_ATTEMPTS = 3;
const POLL_INTERVAL = 3000;
const POLL_TIMEOUT = 5 * 60 * 1000;

const emit = defineEmits<{ (e: 'done', sslId: number): void }>();

const open = ref(false);
const running = ref(false);
const state = ref<'idle' | 'running' | 'success' | 'failed'>('idle');
const attempt = ref(0);
const errorMsg = ref('');
const domainText = ref('');
const phase = ref<'creating' | 'waiting'>('creating');
const sslIdRef = ref(0);
const logRef = ref();
const hasLog = computed(() => sslIdRef.value > 0);

const openLog = () => {
    logRef.value?.acceptParams({ id: sslIdRef.value, type: 'ssl', tail: running.value });
};

let websiteId = 0;
let acmeAccountId = 0;
let sslId = 0;
let cancelled = false;

const statusText = computed(() => {
    if (state.value === 'success') return i18n.global.t('website.newSSLSuccess');
    if (state.value === 'failed') return i18n.global.t('website.newSSLFailed');
    return phase.value === 'creating'
        ? i18n.global.t('website.newSSLApplying')
        : i18n.global.t('website.newSSLWaiting');
});

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
const isIP = (value: string) =>
    /^(\d{1,3}\.){3}\d{1,3}$/.test(value) || (value.includes(':') && /^[0-9a-f:]+$/i.test(value));

const acceptParams = (params: { websiteId: number; acmeAccountId: number }) => {
    websiteId = params.websiteId;
    acmeAccountId = params.acmeAccountId;
    sslId = 0;
    sslIdRef.value = 0;
    open.value = true;
    start(true);
};

const createCert = async () => {
    const res = await listDomains(websiteId);
    const names: string[] = [];
    for (const item of res.data || []) {
        if (item.domain && !names.includes(item.domain)) {
            names.push(item.domain);
        }
    }
    if (names.length === 0) {
        throw new Error(i18n.global.t('website.newSSLNoDomain'));
    }
    domainText.value = names.join(', ');
    const created = await createSSL({
        primaryDomain: names[0],
        otherDomains: names.slice(1).join('\n'),
        provider: 'http',
        acmeAccountId: acmeAccountId,
        dnsAccountId: 0,
        description: '',
        isIP: isIP(names[0]),
        keyType: 'EC256',
        autoRenew: true,
    });
    return (created.data as any)?.id as number;
};

// waits until the certificate leaves the init/applying states
const waitForResult = async () => {
    const deadline = Date.now() + POLL_TIMEOUT;
    while (!cancelled && Date.now() < deadline) {
        await sleep(POLL_INTERVAL);
        const res = await getSSL(sslId);
        const ssl = res.data;
        if (ssl.status === 'ready' && ssl.pem) {
            return;
        }
        if (ssl.status === 'error' || ssl.status === 'applyError') {
            throw new Error(ssl.message || i18n.global.t('website.newSSLFailed'));
        }
    }
    if (cancelled) {
        throw new Error('cancelled');
    }
    throw new Error(i18n.global.t('website.newSSLTimeout'));
};

const start = async (reset: boolean) => {
    if (reset) {
        attempt.value = 0;
    }
    cancelled = false;
    running.value = true;
    state.value = 'running';
    errorMsg.value = '';

    while (attempt.value < MAX_ATTEMPTS && !cancelled) {
        attempt.value++;
        try {
            phase.value = 'creating';
            if (sslId === 0) {
                sslId = await createCert();
                sslIdRef.value = sslId;
            } else {
                await obtainSSL({ ID: sslId });
            }
            phase.value = 'waiting';
            await waitForResult();
            running.value = false;
            state.value = 'success';
            emit('done', sslId);
            open.value = false;
            return;
        } catch (e: any) {
            errorMsg.value = e?.message || String(e);
            if (e?.message === 'cancelled') break;
        }
    }
    running.value = false;
    state.value = 'failed';
};

const handleClose = async () => {
    if (running.value) return;
    // drop a certificate that never became ready so failed attempts don't pile up
    if (state.value === 'failed' && sslId > 0) {
        try {
            await deleteSSL({ ids: [sslId] } as any);
        } catch (e) {
            // ignore, user can remove it from the certificate list
        }
        sslId = 0;
        sslIdRef.value = 0;
    }
    open.value = false;
};

onBeforeUnmount(() => {
    cancelled = true;
});

defineExpose({ acceptParams });
</script>
