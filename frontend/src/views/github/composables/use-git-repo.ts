// fork: shared logic of the repository source forms (create form, repository settings, GitHub project edit).
import { computed, ref, Ref } from 'vue';
import i18n from '@/lang';
import { Github } from '@/api/interface/github';
import { checkGithubRepo, createGithubKey, listGithubKeyOptions } from '@/api/modules/github';
import { MsgError, MsgSuccess } from '@/utils/message';

export const emptySource = (): Github.Source => ({
    cloneURL: '',
    branch: '',
    authMode: 'public',
    keyID: 0,
    tokenUser: '',
    token: '',
});

export const isSSHURL = (value: string): boolean => {
    const text = value.trim();
    return text.startsWith('git@') || text.startsWith('ssh://');
};
export const isHTTPSURL = (value: string): boolean => value.trim().startsWith('https://');

// owner/repo of whatever was typed: owner/repo, git@github.com:owner/repo.git or an https URL.
export const repoFromInput = (value: string): string => {
    let text = value
        .trim()
        .replace(/\/$/, '')
        .replace(/\.git$/, '');
    if (text.startsWith('git@')) {
        text = text.substring(text.indexOf(':') + 1);
    } else if (text.includes('://')) {
        text = text.substring(text.indexOf('://') + 3);
        text = text.substring(text.indexOf('/') + 1);
    }
    return /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(text) ? text : '';
};

export const buildCloneURL = (source: Github.Source): string => {
    const value = source.cloneURL.trim();
    if (/[\s'"`;|&$<>\\]/.test(value)) {
        return '';
    }
    if (value.includes('://') || value.startsWith('git@')) {
        return value;
    }
    const name = repoFromInput(value);
    if (!name) {
        return '';
    }
    return source.authMode === 'ssh' ? `git@github.com:${name}.git` : `https://github.com/${name}.git`;
};

// The key mode the project API understands.
export const keyModeOf = (source: Github.Source): Github.KeyMode => {
    if (source.authMode === 'token') return 'token';
    if (source.authMode === 'ssh') return source.keyID ? 'custom' : 'shared';
    return 'none';
};

// Returns an error message, or '' when the source is usable. A stored token (edit) may stay empty.
export const sourceError = (source: Github.Source, tokenStored = false): string => {
    const url = buildCloneURL(source);
    if (!url) return i18n.global.t('github.repoHelper');
    if (source.authMode === 'token') {
        if (!isHTTPSURL(url)) return i18n.global.t('github.needHTTPSForToken');
        if (!source.token.trim() && !tokenStored) return i18n.global.t('github.tokenHelper');
    }
    if (source.authMode === 'public' && isSSHURL(url)) return i18n.global.t('github.keyNoneHelper');
    if (source.authMode === 'ssh' && !isSSHURL(url)) return i18n.global.t('github.needSSHForKey');
    if (!source.branch.trim()) return i18n.global.t('github.selectBranch');
    return '';
};

export interface GitRepoOptions {
    // Keep the branch that is set even when the remote list does not contain it (edit forms).
    keepBranch?: boolean;
    projectID?: Ref<number | undefined>;
    tokenStored?: Ref<boolean>;
}

export const useGitRepo = (source: Github.Source, options: GitRepoOptions = {}) => {
    const keys = ref<Github.KeyOption[]>([]);
    const branches = ref<string[]>([]);
    const checkResult = ref<Github.RepoCheckInfo>();
    const checking = ref(false);
    const creatingKey = ref(false);

    const selectedKey = computed(() =>
        source.authMode === 'ssh' ? keys.value.find((item) => item.id === source.keyID) : undefined,
    );
    const branchOptions = computed(() => {
        const list = [...branches.value];
        if (source.branch && !list.includes(source.branch)) {
            list.unshift(source.branch);
        }
        return list;
    });
    const githubRepoName = computed(() => repoFromInput(source.cloneURL));

    const loadKeys = async () => {
        const res = await listGithubKeyOptions();
        keys.value = res.data || [];
        if (source.authMode === 'ssh' && !source.keyID && keys.value.length > 0) {
            source.keyID = keys.value[0].id;
        }
    };

    const resetCheck = () => {
        checkResult.value = undefined;
    };
    const onUrlChange = () => {
        checkResult.value = undefined;
        branches.value = [];
    };
    const onAuthChange = () => {
        checkResult.value = undefined;
        if (source.authMode === 'ssh' && !source.keyID && keys.value.length > 0) {
            source.keyID = keys.value[0].id;
        }
    };

    // Reaches the repository with the chosen credentials and fills the branch list.
    const check = async (quiet = false): Promise<boolean> => {
        const cloneURL = buildCloneURL(source);
        const message = sourceError({ ...source, branch: source.branch || 'x' }, options.tokenStored?.value);
        if (!cloneURL || message) {
            if (!quiet) MsgError(message || i18n.global.t('github.repoHelper'));
            return false;
        }
        checking.value = true;
        try {
            const res = await checkGithubRepo({
                cloneURL: cloneURL,
                keyMode: keyModeOf(source),
                keyID: source.authMode === 'ssh' ? source.keyID : 0,
                tokenUser: source.tokenUser,
                token: source.token,
                projectID: options.projectID?.value,
            });
            checkResult.value = res.data;
            branches.value = res.data.branches || [];
            if (res.data.ok && (!source.branch || (!options.keepBranch && !branches.value.includes(source.branch)))) {
                source.branch = res.data.defaultBranch || branches.value[0] || '';
            }
            return res.data.ok;
        } finally {
            checking.value = false;
        }
    };

    const generateKey = async () => {
        const base = githubRepoName.value.split('/')[1] || 'key';
        const name = `${base}_${Math.floor(10000 + Math.random() * 90000)}`;
        creatingKey.value = true;
        try {
            const res = await createGithubKey(name, 'shared');
            await loadKeys();
            source.keyID = res.data.id;
            checkResult.value = undefined;
            MsgSuccess(i18n.global.t('github.keyCreated'));
        } finally {
            creatingKey.value = false;
        }
    };

    return {
        keys,
        branches,
        checkResult,
        checking,
        creatingKey,
        selectedKey,
        branchOptions,
        githubRepoName,
        loadKeys,
        resetCheck,
        onUrlChange,
        onAuthChange,
        check,
        generateKey,
    };
};
