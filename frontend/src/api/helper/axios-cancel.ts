import axios, { AxiosRequestConfig, Canceler } from 'axios';
import qs from 'qs';

// * Map that stores each request identifier and its cancel function
let pendingMap = new Map<string, Canceler>();

// * Serialize params
export const getPendingUrl = (config: AxiosRequestConfig) =>
    [config.method, config.url, qs.stringify(config.data), qs.stringify(config.params)].join('&');

export class AxiosCanceler {
    /**
     * @description: add a request
     * @param {Object} config
     * @return void
     */
    addPending(config: AxiosRequestConfig) {
        // * Before a request starts, check and cancel the previous request
        this.removePending(config);
        const url = getPendingUrl(config);
        config.cancelToken =
            config.cancelToken ||
            new axios.CancelToken((cancel) => {
                if (!pendingMap.has(url)) {
                    // If the request is not in pending, add it
                    pendingMap.set(url, cancel);
                }
            });
    }

    /**
     * @description: remove a request
     * @param {Object} config
     */
    removePending(config: AxiosRequestConfig) {
        const url = getPendingUrl(config);

        if (pendingMap.has(url)) {
            // If the request identifier exists in pending, cancel the request and remove it
            const cancel = pendingMap.get(url);
            typeof cancel === 'function' && cancel();
            pendingMap.delete(url);
        }
    }

    /**
     * @description: clear all pending requests
     */
    removeAllPending() {
        pendingMap.forEach((cancel) => {
            typeof cancel === 'function' && cancel();
        });
        pendingMap.clear();
    }

    /**
     * @description: reset
     */
    reset(): void {
        pendingMap = new Map<string, Canceler>();
    }
}
