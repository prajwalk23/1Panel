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

export const domainValue = (websiteID: number, domain: string): string => {
    return websiteID ? `${websiteID}|${domain || ''}` : '';
};

export const parseDomainValue = (value: string): { websiteID: number; domain: string } => {
    if (!value) {
        return { websiteID: 0, domain: '' };
    }
    const idx = value.indexOf('|');
    return { websiteID: Number(value.substring(0, idx)) || 0, domain: value.substring(idx + 1) };
};
