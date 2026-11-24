<template>
    <DialogPro v-model="open" :title="$t('setting.upgradePlanTitle')" @close="handleClose">
        <div v-if="version" class="mb-3">{{ $t('setting.upgradeAvailable', [version]) }}</div>
        <div v-if="releaseNote" class="mb-3">
            <div class="whats-new-title">{{ $t('setting.upgradeWhatsNew') }}</div>
            <div class="whats-new panel-MdEditor">
                <MarkDownEditor :content="releaseNote" />
            </div>
        </div>
        <el-alert v-if="plan.mode !== 'none'" type="info" :closable="false" class="mb-3" :title="planText" />

        <div v-if="!scheduling">
            <div class="plan-actions">
                <el-button type="primary" @click="onUpgradeNow">{{ $t('setting.upgradeNow') }}</el-button>
                <el-button @click="scheduling = true">{{ $t('setting.upgradeScheduleBtn') }}</el-button>
                <el-button @click="onIgnore">{{ $t('setting.upgradeIgnoreBtn') }}</el-button>
            </div>
            <div class="input-help">{{ $t('setting.upgradeIgnoreHelper') }}</div>
        </div>

        <div v-else>
            <el-radio-group v-model="mode" class="mb-3">
                <el-radio value="once">{{ $t('setting.upgradeScheduleOnce') }}</el-radio>
                <el-radio value="daily">{{ $t('setting.upgradeScheduleDaily') }}</el-radio>
            </el-radio-group>
            <div>
                <el-date-picker
                    v-if="mode === 'once'"
                    v-model="onceTime"
                    type="datetime"
                    format="YYYY-MM-DD HH:mm"
                    value-format="YYYY-MM-DD HH:mm"
                    :disabled-date="disablePast"
                    :clearable="false"
                />
                <el-time-picker v-else v-model="dailyTime" format="HH:mm" value-format="HH:mm" :clearable="false" />
            </div>
            <div class="input-help">{{ $t('setting.upgradeScheduleHelper') }}</div>
        </div>

        <template #footer>
            <span class="dialog-footer">
                <el-button v-if="plan.mode !== 'none'" @click="onRemove">
                    {{ $t('setting.upgradePlanRemove') }}
                </el-button>
                <el-button @click="handleClose">{{ $t('commons.button.cancel') }}</el-button>
                <el-button v-if="scheduling" type="primary" :disabled="!canSave" @click="onSaveSchedule">
                    {{ $t('commons.button.confirm') }}
                </el-button>
            </span>
        </template>
    </DialogPro>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import MarkDownEditor from '@/components/mkdown-editor/index.vue';
import { updateUpgradePlan } from '@/api/modules/setting';
import { Setting } from '@/api/interface/setting';
import i18n from '@/lang';
import { MsgSuccess } from '@/utils/message';

const emit = defineEmits(['upgrade', 'changed']);

const open = ref(false);
const version = ref('');
const releaseNote = ref('');
const plan = ref<Setting.UpgradePlan>({ mode: 'none', time: '' });
const scheduling = ref(false);
const mode = ref('daily');
const onceTime = ref('');
const dailyTime = ref('03:00');

interface DialogProps {
    version: string;
    plan: Setting.UpgradePlan;
    releaseNote?: string;
}

const acceptParams = (params: DialogProps) => {
    version.value = params.version || '';
    releaseNote.value = params.releaseNote || '';
    plan.value = params.plan || { mode: 'none', time: '' };
    scheduling.value = false;
    mode.value = plan.value.mode === 'once' ? 'once' : 'daily';
    onceTime.value = plan.value.mode === 'once' ? plan.value.time : '';
    dailyTime.value = plan.value.mode === 'daily' ? plan.value.time : '03:00';
    open.value = true;
};

const planText = computed(() => {
    switch (plan.value.mode) {
        case 'once':
            return i18n.global.t('setting.upgradePlanOnce', [plan.value.time]);
        case 'daily':
            return i18n.global.t('setting.upgradePlanDaily', [plan.value.time]);
        case 'ignore':
            return i18n.global.t('setting.upgradePlanIgnore');
        default:
            return '';
    }
});

const canSave = computed(() => (mode.value === 'once' ? !!onceTime.value : !!dailyTime.value));

const disablePast = (date: Date) => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    return date.getTime() < today.getTime();
};

const handleClose = () => {
    open.value = false;
};

const save = async (planMode: string, time: string, msg: string) => {
    await updateUpgradePlan(planMode, time);
    MsgSuccess(i18n.global.t(msg));
    open.value = false;
    emit('changed');
};

const onUpgradeNow = () => {
    open.value = false;
    emit('upgrade');
};

const onIgnore = () => save('ignore', '', 'setting.upgradePlanSaved');
const onRemove = () => save('none', '', 'setting.upgradePlanRemoved');
const onSaveSchedule = () =>
    save(mode.value, mode.value === 'once' ? onceTime.value : dailyTime.value, 'setting.upgradePlanSaved');

defineExpose({ acceptParams });
</script>

<style lang="scss" scoped>
.plan-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}
.whats-new-title {
    font-weight: 500;
    margin-bottom: 6px;
}
.whats-new {
    max-height: 260px;
    overflow-y: auto;
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    :deep(.md-editor-preview) {
        font-size: 13px;
    }
}
:deep(.md-editor-dark) {
    background-color: var(--panel-main-bg-color-9);
}
</style>
