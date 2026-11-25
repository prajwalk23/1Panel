// fork: API calls of the GitHub section. Agent routes live under /github, the
// fork updater under /core/github.
import http from '@/api';
import { ResPage } from '../interface';
import { Github } from '../interface/github';
import { TimeoutEnum } from '@/enums/http-enum';

export const searchGithubProjects = (params: Github.ProjectSearch) => {
    return http.post<ResPage<Github.ProjectInfo>>(`/github/projects/search`, params);
};
export const createGithubProject = (params: Github.ProjectOperate) => {
    return http.post<Github.ProjectInfo>(`/github/projects`, params, TimeoutEnum.T_60S);
};
export const updateGithubProject = (params: Github.ProjectOperate) => {
    return http.post(`/github/projects/update`, params, TimeoutEnum.T_60S);
};
export const deleteGithubProject = (id: number) => {
    return http.post(`/github/projects/del`, { id: id }, TimeoutEnum.T_60S);
};
export const updateGithubProjectStatus = (id: number, enabled: boolean) => {
    return http.post(`/github/projects/status`, { id: id, enabled: enabled });
};
export const deployGithubProject = (params: Github.ProjectDeploy) => {
    return http.post(`/github/projects/deploy`, params, TimeoutEnum.T_60S);
};
export const loadLinkedGithubProject = (params: Github.LinkedSearch) => {
    return http.post<Github.ProjectInfo>(`/github/projects/linked`, params, TimeoutEnum.T_60S);
};
export const loadGithubSummary = (id: number) => {
    return http.post<Github.ProjectSummary>(`/github/projects/summary`, { id: id });
};
export const searchGithubCommits = (params: Github.CommitSearch) => {
    return http.post<ResPage<Github.CommitInfo>>(`/github/projects/commits`, params, TimeoutEnum.T_60S);
};
export const cancelGithubDeploy = (id: number) => {
    return http.post(`/github/projects/cancel`, { id: id });
};
export const refreshGithubWebhookToken = (id: number) => {
    return http.post<Github.ProjectSecret>(`/github/projects/webhook/token`, { id: id });
};
export const configureGithubWebhook = (params: Github.WebhookConfig) => {
    return http.post<Github.ProjectSecret>(`/github/projects/webhook/config`, params, TimeoutEnum.T_60S);
};
export const loadGithubWebhook = (id: number) => {
    return http.post<Github.WebhookInfo>(`/github/projects/webhook`, { id: id });
};

export const listGithubScripts = (projectID: number) => {
    return http.post<Array<Github.ScriptInfo>>(`/github/scripts/list`, { projectID: projectID });
};
export const loadGithubDefaultScript = (projectID: number) => {
    return http.post<string>(`/github/scripts/default`, { projectID: projectID });
};
export const createGithubScript = (params: Github.ScriptOperate) => {
    return http.post<Github.ScriptInfo>(`/github/scripts`, params);
};
export const updateGithubScript = (params: Github.ScriptOperate) => {
    return http.post(`/github/scripts/update`, params);
};
export const deleteGithubScript = (id: number) => {
    return http.post(`/github/scripts/del`, { id: id });
};
export const listGithubSystemUsers = () => {
    return http.get<Array<string>>(`/github/users`);
};

export const searchGithubLogs = (params: Github.LogSearch) => {
    return http.post<ResPage<Github.LogInfo>>(`/github/logs/search`, params);
};
export const cleanGithubLogs = (projectID: number, trigger = '') => {
    return http.post(`/github/logs/clean`, { projectID: projectID, trigger: trigger });
};

export const searchGithubKeys = (params: Github.KeySearch) => {
    return http.post<ResPage<Github.KeyInfo>>(`/github/keys/search`, params);
};
export const createGithubKey = (name: string, type: string) => {
    return http.post<Github.KeyInfo>(`/github/keys`, { name: name, type: type });
};
export const importGithubKey = (name: string, privateKey: string) => {
    return http.post<Github.KeyInfo>(`/github/keys/import`, { name: name, privateKey: privateKey });
};
export const regenerateGithubKey = (id: number) => {
    return http.post<Github.KeyInfo>(`/github/keys/regenerate`, { id: id });
};
export const deleteGithubKey = (id: number) => {
    return http.post(`/github/keys/del`, { id: id });
};
export const listGithubKeyOptions = () => {
    return http.get<Array<Github.KeyOption>>(`/github/keys/options`);
};

export const loadGithubSetting = () => {
    return http.get<Github.SettingInfo>(`/github/settings`);
};
export const updateGithubSetting = (params: Github.Setting) => {
    return http.post(`/github/settings/update`, params, TimeoutEnum.T_60S);
};
export const checkGithubRepo = (params: Github.RepoCheck) => {
    return http.post<Github.RepoCheckInfo>(`/github/repos/check`, params, TimeoutEnum.T_60S);
};
