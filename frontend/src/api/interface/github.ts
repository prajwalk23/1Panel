// fork: types of the GitHub section.
import { ReqPage } from '.';

export namespace Github {
    export type KeyMode = 'project' | 'shared' | 'custom' | 'none';
    export type UpdateStrategy = 'ff-only' | 'reset';

    export interface ProjectSearch extends ReqPage {
        info: string;
    }
    export interface ProjectOperate {
        id?: number;
        name: string;
        cloneURL: string;
        repo: string;
        branch: string;
        projectPath: string;
        deployCommand?: string; // seeds the default script when creating
        updateStrategy?: UpdateStrategy;
        timeout: number;
        keyMode: KeyMode;
        keyID: number;
        enabled: boolean;
    }
    export interface ProjectInfo extends ProjectOperate {
        id: number;
        webhookEnabled: boolean;
        webhookScriptID: number;
        createdAt: string;
        keyName: string;
        publicKey: string;
        lastStatus: string;
        lastRunAt: string;
        lastCommit: string;
        running: boolean;
        webhookURL: string;
        webhookHTTPS: boolean;
    }
    export interface ProjectSecret {
        id: number;
        secret: string;
        webhookURL: string;
        publicKey: string;
    }
    export interface WebhookInfo {
        scriptID: number;
        scriptName: string;
        url: string;
        https: boolean;
        contentType: string;
        events: string;
        enabled: boolean;
        proxyPass: string;
        listener: string;
        message: string;
    }

    export interface LogSearch extends ReqPage {
        projectID?: number;
        status?: string;
    }
    export interface LogInfo {
        id: number;
        createdAt: string;
        projectID: number;
        projectName: string;
        trigger: string;
        deliveryID: string;
        ref: string;
        commit: string;
        pusher: string;
        status: string;
        taskID: string;
        message: string;
        startedAt: string;
        finishedAt: string;
    }

    export interface KeySearch extends ReqPage {
        info: string;
    }
    export interface KeyInfo {
        id: number;
        createdAt: string;
        name: string;
        type: string;
        publicKey: string;
        fingerprint: string;
        isShared: boolean;
        usedBy: string[];
    }
    export interface KeyOption {
        id: number;
        name: string;
        type: string;
        fingerprint: string;
        publicKey: string;
    }

    export interface ScriptInfo {
        id: number;
        createdAt: string;
        projectID: number;
        name: string;
        content: string;
        runAsUser: string;
        isDefault: boolean;
        linked: boolean;
    }
    export interface ScriptOperate {
        id?: number;
        projectID: number;
        name: string;
        content: string;
        runAsUser: string;
    }
    export interface WebhookConfig {
        id: number;
        enabled: boolean;
        scriptID: number;
    }

    export interface Setting {
        webhookPort: number;
        webhookBind: string;
        defaultShell: string;
        logRetention: number;
        webhookDomain: string;
        sharedKeyID: number;
    }
    export interface SettingInfo extends Setting {
        listenerRunning: boolean;
        listenerAddr: string;
        listenerError: string;
        gitInstalled: boolean;
    }
    export interface RepoCheck {
        cloneURL: string;
        keyMode: KeyMode;
        keyID: number;
    }
    export interface RepoCheckInfo {
        ok: boolean;
        message: string;
        defaultBranch: string;
        branches: string[];
    }

    export interface ForkInfo {
        isForkBuild: boolean;
        panelVersion: string;
        forkVersion: string;
        forkRepo: string;
        latestVersion: string;
        latestBase: string;
        hasUpdate: boolean;
        baseChange: boolean;
        releaseName: string;
        releaseNote: string;
        releaseURL: string;
        publishedAt: string;
        arch: string;
        assetReady: boolean;
        systemStatus: string;
    }
}
