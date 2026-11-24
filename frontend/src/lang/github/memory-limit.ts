// fork: messages of Settings > Panel > Memory limit (key `memoryLimit`).
export const memoryLimitEn = {
    title: 'Memory limit',
    help: 'Soft limit for the Go heap of the core and agent processes. When usage nears it the garbage collector works harder instead of letting memory grow. It is not a hard cap.',
    envOverride: 'The GOMEMLIMIT environment variable is set on the core process and overrides this value there.',
    usage: 'Current usage (RSS)',
    core: 'Core',
    agent: 'Agent',
    heap: 'heap',
    notRunning: 'not running',
    limit: 'Limit (GB)',
    range: 'Between {0} and {1} GB. Default {2} GB.',
    appliedHelper: 'Applied immediately. The agent picks it up within one minute.',
    over: 'Above the limit',
};
