<template>
    <div class="runtime-git">
        <el-form-item :label="$t('runtime.gitSource')">
            <el-switch v-model="enabled" />
            <span class="input-help">{{ $t('runtime.gitSourceHelper') }}</span>
        </el-form-item>
        <template v-if="enabled">
            <el-form-item :label="$t('runtime.gitCloneURL')" prop="gitCloneURL" :rules="urlRules">
                <el-input v-model.trim="runtime.gitCloneURL" :placeholder="$t('runtime.gitCloneURLHelper')" />
            </el-form-item>
            <el-row :gutter="20">
                <el-col :span="12" :xs="24" :sm="24" :md="12">
                    <el-form-item :label="$t('runtime.gitBranch')" prop="gitBranch">
                        <el-input v-model.trim="runtime.gitBranch" placeholder="main" :disabled="mode === 'edit'" />
                    </el-form-item>
                </el-col>
                <el-col :span="12" :xs="24" :sm="24" :md="12">
                    <el-form-item :label="$t('runtime.gitKey')" prop="gitKeyID">
                        <el-select v-model="runtime.gitKeyID" clearable>
                            <el-option :label="$t('runtime.gitKeyShared')" :value="0" />
                            <el-option
                                v-for="key in keys"
                                :key="key.id"
                                :label="`${key.name} (${key.fingerprint})`"
                                :value="key.id"
                            />
                        </el-select>
                        <span class="input-help">{{ $t('runtime.gitKeyHelper') }}</span>
                    </el-form-item>
                </el-col>
            </el-row>
            <el-form-item :label="$t('runtime.gitUpdateStrategy')" prop="gitUpdateStrategy">
                <el-radio-group v-model="runtime.gitUpdateStrategy">
                    <el-radio value="ff-only">{{ $t('runtime.gitFFOnly') }}</el-radio>
                    <el-radio value="reset">{{ $t('runtime.gitReset') }}</el-radio>
                </el-radio-group>
                <span class="input-help">{{ $t('runtime.gitUpdateStrategyHelper') }}</span>
            </el-form-item>
        </template>
    </div>
</template>

<script lang="ts" setup>
import { Github } from '@/api/interface/github';
import { listGithubKeyOptions } from '@/api/modules/github';
import { computed, onMounted, ref } from 'vue';
import { useVModel } from '@vueuse/core';
import i18n from '@/lang';

const props = defineProps({
    mode: {
        type: String,
        required: true,
    },
    modelValue: {
        type: Object,
        required: true,
    },
});
const emit = defineEmits(['update:modelValue']);
const runtime = useVModel(props, 'modelValue', emit);

const keys = ref<Github.KeyOption[]>([]);

const enabled = computed({
    get: () => runtime.value.gitMode === 'git',
    set: (val: boolean) => {
        runtime.value.gitMode = val ? 'git' : 'none';
        if (val) {
            if (!runtime.value.gitBranch) {
                runtime.value.gitBranch = 'main';
            }
            if (!runtime.value.gitUpdateStrategy) {
                runtime.value.gitUpdateStrategy = 'ff-only';
            }
            if (runtime.value.gitKeyID === undefined) {
                runtime.value.gitKeyID = 0;
            }
        }
    },
});

const urlRules = computed(() => {
    if (!enabled.value) {
        return [];
    }
    return [{ required: true, message: i18n.global.t('runtime.gitCloneURLRequired'), trigger: 'blur' }];
});

onMounted(() => {
    if (!runtime.value.gitMode) {
        runtime.value.gitMode = 'none';
    }
    listGithubKeyOptions()
        .then((res) => {
            keys.value = res.data || [];
        })
        .catch(() => {
            keys.value = [];
        });
});
</script>
