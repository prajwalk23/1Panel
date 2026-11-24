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

export const getPanelMemoryLimit = () => {
    return http.get<MemoryLimit.Info>(`/core/settings/memory-limit`);
};

export const updatePanelMemoryLimit = (limitMiB: number) => {
    return http.post(`/core/settings/memory-limit`, { limitMiB: limitMiB });
};
