import { ElMessageBox } from 'element-plus';
import i18n from '@/lang';
import { OperateRuntime, updateRemark } from '@/api/modules/runtime';
import { Ref } from 'vue';
import { MsgError, MsgSuccess } from '@/utils/message';
import { Runtime } from '@/api/interface/runtime';
import { createWebsite } from '@/api/modules/website';
import { newUUID } from '@/utils/id';

export const operateRuntime = async (operate: string, ID: number, loading: Ref<boolean>, search: () => void) => {
    try {
        const action = await ElMessageBox.confirm(
            i18n.global.t('runtime.operatorHelper', [i18n.global.t('commons.operate.' + operate)]),
            i18n.global.t('commons.operate.' + operate),
            {
                confirmButtonText: i18n.global.t('commons.button.confirm'),
                cancelButtonText: i18n.global.t('commons.button.cancel'),
                type: 'info',
            },
        );

        if (action === 'confirm') {
            loading.value = true;
            await OperateRuntime({ operate: operate, ID: ID });
            search();
        }
    } catch (error) {
    } finally {
        loading.value = false;
    }
};

// fields of the optional "Create Website" section of the runtime create forms
export const initRuntimeWebsite = () => ({
    createWebsite: false,
    domains: [] as any[],
    alias: '',
    webSiteGroupId: 0,
    websitePort: 0,
});

export const checkRuntimeWebsite = (runtime: Runtime.RuntimeCreate): boolean => {
    if (!runtime.createWebsite) {
        return true;
    }
    if (!runtime.websitePort) {
        MsgError(i18n.global.t('website.createWebsiteNoPort'));
        return false;
    }
    return true;
};

// runs after the runtime row exists; never throws, the runtime is already created at this point
export const createWebsiteForRuntime = async (runtime: Runtime.RuntimeCreate, created: Runtime.Runtime) => {
    if (!runtime.createWebsite || !created?.id) {
        return;
    }
    try {
        await createWebsite({
            type: 'runtime',
            alias: runtime.alias,
            remark: '',
            appType: 'installed',
            appInstallId: undefined,
            webSiteGroupId: runtime.webSiteGroupId,
            proxy: '',
            proxyType: 'tcp',
            runtimeID: created.id,
            port: runtime.websitePort,
            ftpUser: '',
            ftpPassword: '',
            taskID: newUUID(),
            enableSSL: false,
            IPV6: false,
            domains: runtime.domains,
            name: '',
            algorithm: '',
            servers: [],
            gitMode: 'none',
        } as any);
    } catch (error) {
        MsgError(i18n.global.t('website.createWebsiteFailed'));
    }
};

export const updateRuntimeRemark = async (row: Runtime.Runtime, bulr: Function) => {
    bulr();
    if (row.remark && row.remark.length > 128) {
        MsgError(i18n.global.t('commons.rule.length128Err'));
        return;
    }
    try {
        await updateRemark({
            id: row.id,
            remark: row.remark,
        }).then(() => {
            MsgSuccess(i18n.global.t('commons.msg.updateSuccess'));
        });
    } catch (error) {}
};
