// Browser-side bookkeeping for the "new version available" prompt.
export const IGNORED_KEY = 'upgrade-ignored-version';
export const PROMPTED_PREFIX = 'upgrade-prompted-';

// Forget every remembered prompt/ignore decision. Called once an upgrade is accepted so the
// next version check starts clean; it does not look at any version.
export const clearUpgradePromptState = () => {
    try {
        localStorage.removeItem(IGNORED_KEY);
        Object.keys(sessionStorage)
            .filter((key) => key.startsWith(PROMPTED_PREFIX))
            .forEach((key) => sessionStorage.removeItem(key));
    } catch (e) {
        // storage blocked: nothing to clear
    }
};
