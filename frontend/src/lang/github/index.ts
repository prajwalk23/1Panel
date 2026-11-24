// fork: merges the GitHub section messages into a locale. Locales without
// their own file fall back to English through vue-i18n's fallbackLocale.
import en from './en';
import zh from './zh';
import { memoryLimitEn, memoryLimitZh } from './memory-limit';

type LocaleMessage = Record<string, unknown>;

const githubMessages: Record<string, LocaleMessage> = { en, zh };
const memoryLimitMessages: Record<string, LocaleMessage> = { en: memoryLimitEn, zh: memoryLimitZh };

export const withGithubMessages = (locale: string, messages: LocaleMessage): LocaleMessage => {
    const extra = githubMessages[locale];
    const memoryLimit = memoryLimitMessages[locale];
    const merged: LocaleMessage = extra ? { ...messages, github: extra } : messages;
    return memoryLimit ? { ...merged, memoryLimit } : merged;
};
