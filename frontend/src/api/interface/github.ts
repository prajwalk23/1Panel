// fork: types of the GitHub section.
import { ReqPage } from '.';

export namespace Github {
    // none = public repository, token = https + access token, the rest use an SSH key
    export type KeyMode = 'project' | 'shared' | 'custom' | 'none' | 'token';
    export type UpdateStrategy = 'ff-only' | 'reset';
    // What the create form asks for: public | token | ssh
    export type AuthMode = 'public' | 'token' | 'ssh';

    export interface ProjectSearch extends ReqPage {
        info: string;
        standalone?: boolean; // only projects that are not linked to a website or runtime
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
        tokenUser?: string;
        token?: string; // empty on update keeps the stored token
        keepRecords?: number;
        ignorePaths?: string;
        enabled: boolean;
    }
    export interface ProjectInfo extends ProjectOperate {
        id: number;
        websiteID: number;
        runtimeID: number;
        hasToken: boolean;
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
        webhookURL: string;
        publicKey: string;
    }
    export interface LinkedSearch {
        websiteID?: number;
        runtimeID?: number;
    }
    export interface ProjectDeploy {
        id: number;
        scriptID?: number;
        commit?: string;
        rollback?: boolean;
    }
    export interface CommitSearch extends ReqPage {
        id: number;
        branch: string;
    }
    export interface CommitInfo {
        commit: string;
        author: string;
        message: string;
        time: string;
        active: boolean;
    }
    export interface ProjectSummary {
        project: ProjectInfo;
        commit: string;
        commitMessage: string;
        commitAuthor: string;
        deployedAt: string;
        duration: number;
    }
    export interface WebhookInfo {
        enabled: boolean;
        scriptID: number;
        scriptName: string;
        url: string;
        path: string; // /hook?access_key=..., the host part comes from the panel address
        bindDomain: boolean;
        https: boolean;
        contentType: string;
        events: string;
    }

    export interface LogSearch extends ReqPage {
        projectID?: number;
        status?: string;
        trigger?: string;
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
        commitMessage: string;
        commitAuthor: string;
        duration: number;
        active: boolean;
        rollbackable: boolean;
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
        defaultShell: string;
        logRetention: number;
        sharedKeyID: number;
    }
    export interface SettingInfo extends Setting {
        webhookBase: string;
        bindDomain: boolean;
        gitInstalled: boolean;
    }
    export interface RepoCheck {
        cloneURL: string;
        keyMode: KeyMode;
        keyID: number;
        tokenUser?: string;
        token?: string;
        projectID?: number; // reuse the stored token of this project when token is empty
    }
    export interface RepoCheckInfo {
        ok: boolean;
        message: string;
        defaultBranch: string;
        branches: string[];
    }

    // The source of a repository a create form collects (shared by GitHub, website and runtime create).
    export interface Source {
        cloneURL: string;
        branch: string;
        authMode: AuthMode;
        keyID: number;
        tokenUser: string;
        token: string;
    }
}
