// fork: panel memory limit (core settings endpoints).
import http from '@/api';

export namespace MemoryLimit {
    export interface ProcessMemory {
        running: boolean;
        rssMiB: number;
        heapMiB: number;
    }
    export interface Info {
        limitMiB: number;
        defaultMiB: number;
        minMiB: number;
        maxMiB: number;
        envOverride: boolean;
        core: ProcessMemory;
        agent: ProcessMemory;
    }
}

// The backend works in MiB; the UI shows GB (1 GB = 1024 MiB here, as the OS reports memory).
export const mibToGB = (mib: number): number => Math.round((mib / 1024) * 100) / 100;
export const gbToMib = (gb: number): number => Math.round(gb * 1024);
export const formatGB = (mib: number): string => `${mibToGB(mib)} GB`;

export const getPanelMemoryLimit = () => {
    return http.get<MemoryLimit.Info>(`/core/settings/memory-limit`);
};

export const updatePanelMemoryLimit = (limitMiB: number) => {
    return http.post(`/core/settings/memory-limit`, { limitMiB: limitMiB });
};
