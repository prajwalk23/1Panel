<template>
    <div>
        <LayoutContent v-loading="loading" :title="$t('github.keys')">
            <template #leftToolBar>
                <el-button v-permission v-node-admin type="primary" @click="openCreate('generate')">
                    {{ $t('github.generateKey') }}
                </el-button>
                <el-button v-permission v-node-admin @click="openCreate('import')">
                    {{ $t('github.importKey') }}
                </el-button>
            </template>
            <template #rightToolBar>
                <TableSearch @search="search()" v-model:searchName="searchName" />
                <TableRefresh @search="search()" />
            </template>
            <template #main>
                <ComplexTable
                    :cache-pagination="false"
                    :pagination-config="paginationConfig"
                    @search="search"
                    :data="data"
                    row-key="id"
                >
                    <el-table-column :label="$t('github.keyName')" prop="name" min-width="140" show-overflow-tooltip>
                        <template #default="{ row }">
                            <span>{{ row.name }}</span>
                            <el-tag v-if="row.isShared" size="small" type="success" class="ml-2">
                                {{ $t('github.defaultShared') }}
                            </el-tag>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.keyType')" min-width="90">
                        <template #default="{ row }">
                            <el-tag type="info">{{ typeLabel(row.type) }}</el-tag>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.fingerprint')" min-width="200" show-overflow-tooltip>
                        <template #default="{ row }">
                            <span class="font-mono text-xs">{{ row.fingerprint }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.publicKey')" min-width="120">
                        <template #default="{ row }">
                            <CopyButton :content="row.publicKey" :is-icon="false" />
                        </template>
                    </el-table-column>
                    <el-table-column :label="$t('github.usedBy')" min-width="140" show-overflow-tooltip>
                        <template #default="{ row }">
                            {{ row.usedBy && row.usedBy.length ? row.usedBy.join(', ') : $t('github.unused') }}
                        </template>
                    </el-table-column>
                    <fu-table-operations width="180px" :buttons="buttons" :label="$t('commons.table.operate')" fix />
                </ComplexTable>
            </template>
        </LayoutContent>

        <DialogPro v-model="createVisible" :title="createTitle" size="normal">
            <el-form ref="formRef" label-position="top" :model="createForm" :rules="rules" v-loading="createLoading">
                <el-alert
                    v-if="createMode === 'generate'"
                    type="info"
                    :closable="false"
                    class="mb-3"
                    :title="$t('github.generateKeyHelper')"
                />
                <el-form-item :label="$t('github.keyName')" prop="name">
                    <el-input v-model.trim="createForm.name" />
                </el-form-item>
                <el-form-item v-if="createMode === 'import'" :label="$t('github.privateKey')" prop="privateKey">
                    <el-input
                        v-model="createForm.privateKey"
                        type="textarea"
                        :rows="8"
                        class="font-mono"
                        placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                    />
                    <span class="input-help">{{ $t('github.privateKeyHelper') }}</span>
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="createVisible = false">{{ $t('commons.button.cancel') }}</el-button>
                <el-button type="primary" :disabled="createLoading" @click="onCreate">
                    {{ $t('commons.button.confirm') }}
                </el-button>
            </template>
        </DialogPro>

        <DialogPro v-model="publicVisible" :title="$t('github.publicKey')" size="normal">
            <p class="mb-2">{{ $t('github.deployKeySteps') }}</p>
            <div class="flex items-start gap-1">
                <el-input :model-value="publicKeyText" type="textarea" :rows="4" readonly class="font-mono" />
                <CopyButton :content="publicKeyText" />
            </div>
            <template #footer>
                <el-button @click="publicVisible = false">{{ $t('commons.button.close') }}</el-button>
            </template>
        </DialogPro>

        <OpDialog ref="opRef" @search="search" @submit="onSubmitDelete()" />
    </div>
</template>

<script lang="ts" setup>
// fork: SSH deploy keys. Only public keys are ever shown.
import { computed, onMounted, reactive, ref } from 'vue';
import { ElForm, ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { Rules } from '@/global/form-rules';
import { MsgSuccess } from '@/utils/message';
import { Github } from '@/api/interface/github';
import {
    createGithubKey,
    deleteGithubKey,
    importGithubKey,
    regenerateGithubKey,
    searchGithubKeys,
} from '@/api/modules/github';

type FormInstance = InstanceType<typeof ElForm>;

const loading = ref(false);
const data = ref<Github.KeyInfo[]>([]);
const searchName = ref('');
const paginationConfig = reactive({ currentPage: 1, pageSize: 20, total: 0 });
const opRef = ref();
const deleteID = ref(0);

const createVisible = ref(false);
const createLoading = ref(false);
const createMode = ref<'generate' | 'import'>('generate');
const createForm = reactive({ name: '', privateKey: '' });
const formRef = ref<FormInstance>();
const createTitle = computed(() =>
    createMode.value === 'generate' ? i18n.global.t('github.generateKey') : i18n.global.t('github.importKey'),
);
const rules = reactive({
    name: [Rules.requiredInput],
    privateKey: [Rules.requiredInput],
});

const publicVisible = ref(false);
const publicKeyText = ref('');

const typeLabel = (type: string) => {
    switch (type) {
        case 'project':
            return i18n.global.t('github.typeProject');
        case 'shared':
            return i18n.global.t('github.typeShared');
        default:
            return i18n.global.t('github.typeCustom');
    }
};

const search = async () => {
    loading.value = true;
    try {
        const res = await searchGithubKeys({
            info: searchName.value,
            page: paginationConfig.currentPage,
            pageSize: paginationConfig.pageSize,
        });
        data.value = res.data.items || [];
        paginationConfig.total = res.data.total;
    } finally {
        loading.value = false;
    }
};

const openCreate = (mode: 'generate' | 'import') => {
    createMode.value = mode;
    createForm.name = '';
    createForm.privateKey = '';
    createVisible.value = true;
    formRef.value?.clearValidate();
};

const showPublicKey = (key: Github.KeyInfo) => {
    publicKeyText.value = key.publicKey;
    publicVisible.value = true;
};

const onCreate = async () => {
    if (!formRef.value) return;
    await formRef.value.validate(async (valid) => {
        if (!valid) return;
        createLoading.value = true;
        try {
            const res =
                createMode.value === 'generate'
                    ? await createGithubKey(createForm.name, 'shared')
                    : await importGithubKey(createForm.name, createForm.privateKey);
            createForm.privateKey = '';
            createVisible.value = false;
            MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
            search();
            showPublicKey(res.data);
        } finally {
            createLoading.value = false;
        }
    });
};

const onRegenerate = (row: Github.KeyInfo) => {
    ElMessageBox.confirm(
        i18n.global.t('github.regenerateKeyHelper', [row.name]),
        i18n.global.t('github.regenerateKey'),
        {
            confirmButtonText: i18n.global.t('commons.button.confirm'),
            cancelButtonText: i18n.global.t('commons.button.cancel'),
            type: 'warning',
        },
    ).then(async () => {
        const res = await regenerateGithubKey(row.id);
        MsgSuccess(i18n.global.t('commons.msg.operationSuccess'));
        search();
        showPublicKey(res.data);
    });
};

const onDelete = (row: Github.KeyInfo) => {
    deleteID.value = row.id;
    opRef.value.acceptParams({
        title: i18n.global.t('commons.button.delete'),
        names: [row.name],
        msg: i18n.global.t('commons.msg.operatorHelper', [
            i18n.global.t('github.keys'),
            i18n.global.t('commons.button.delete'),
        ]),
        api: null,
        params: null,
    });
};

const onSubmitDelete = async () => {
    loading.value = true;
    try {
        await deleteGithubKey(deleteID.value);
        MsgSuccess(i18n.global.t('commons.msg.deleteSuccess'));
    } finally {
        loading.value = false;
    }
    search();
};

const buttons = [
    {
        label: i18n.global.t('github.publicKey'),
        click: showPublicKey,
    },
    {
        label: i18n.global.t('github.regenerateKey'),
        permission: true,
        nodeAdmin: true,
        disabled: (row: Github.KeyInfo) => row.type === 'custom',
        click: onRegenerate,
    },
    {
        label: i18n.global.t('commons.button.delete'),
        permission: true,
        nodeAdmin: true,
        disabled: (row: Github.KeyInfo) => row.usedBy && row.usedBy.length > 0,
        click: onDelete,
    },
];

onMounted(() => {
    search();
});
</script>
