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

export const memoryLimitZh = {
    title: '内存上限',
    help: 'core 与 agent 进程的 Go 堆软上限。接近上限时垃圾回收会更积极，而不是让内存继续增长。它不是硬性限制。',
    envOverride: 'core 进程设置了 GOMEMLIMIT 环境变量，该进程以环境变量为准。',
    usage: '当前使用（RSS）',
    core: 'Core',
    agent: 'Agent',
    heap: '堆',
    notRunning: '未运行',
    limit: '上限 (GB)',
    range: '范围 {0} - {1} GB，默认 {2} GB。',
    appliedHelper: '立即生效，agent 会在一分钟内同步。',
    over: '已超过上限',
};
