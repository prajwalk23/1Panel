// fork: small shared helpers of the GitHub views.
import i18n from '@/lang';

export const statusType = (status: string): '' | 'success' | 'danger' | 'warning' | 'info' => {
    switch (status) {
        case 'success':
            return 'success';
        case 'failed':
            return 'danger';
        case 'running':
        case 'waiting':
            return 'warning';
        case 'canceled':
        case 'skipped':
            return 'info';
    }
    return '';
};

export const statusLabel = (status: string): string => {
    if (!status) {
        return i18n.global.t('github.never');
    }
    const key = 'github.status' + status.charAt(0).toUpperCase() + status.slice(1);
    const label = i18n.global.t(key);
    return label === key ? status : label;
};

export const shortCommit = (commit: string): string => {
    return commit ? commit.substring(0, 7) : '-';
};

export const isZeroTime = (value: string): boolean => {
    return !value || value.startsWith('0001-01-01');
};

export const formatTime = (value: string): string => {
    if (isZeroTime(value)) {
        return '-';
    }
    const date = new Date(value);
    if (isNaN(date.getTime())) {
        return value;
    }
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(
        date.getMinutes(),
    )}:${pad(date.getSeconds())}`;
};

export const formatDuration = (start: string, end: string): string => {
    if (isZeroTime(start) || isZeroTime(end)) {
        return '-';
    }
    const seconds = Math.max(0, Math.round((new Date(end).getTime() - new Date(start).getTime()) / 1000));
    if (seconds < 60) {
        return `${seconds}s`;
    }
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ${seconds % 60}s`;
};

export const formatMillis = (ms: number): string => {
    if (!ms || ms < 0) {
        return '-';
    }
    const seconds = ms / 1000;
    if (seconds < 60) {
        return `${seconds.toFixed(seconds < 10 ? 2 : 0)}s`;
    }
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ${Math.round(seconds % 60)}s`;
};

export const triggerLabel = (trigger: string): string => {
    const map: Record<string, string> = {
        webhook: 'github.triggerWebhook',
        manual: 'github.triggerManual',
        rollback: 'github.triggerRollback',
        create: 'github.triggerCreate',
        boot: 'github.triggerBoot',
    };
    return map[trigger] ? i18n.global.t(map[trigger]) : trigger;
};

// The webhook URL as the admin should paste it: the bound domain when one is set, otherwise the
// address the panel is opened with (the server cannot always know the address the admin uses).
export const webhookDisplayURL = (info?: { url: string; path: string; bindDomain: boolean }): string => {
    if (!info || !info.path) {
        return '';
    }
    return info.bindDomain ? info.url : window.location.origin + info.path;
};
