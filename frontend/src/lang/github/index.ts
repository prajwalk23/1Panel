// fork: merges the GitHub section messages into a locale. Locales without
// their own file fall back to English through vue-i18n's fallbackLocale.
import en from './en';
import zh from './zh';

type LocaleMessage = Record<string, unknown>;

const githubMessages: Record<string, LocaleMessage> = { en, zh };

export const withGithubMessages = (locale: string, messages: LocaleMessage): LocaleMessage => {
    const extra = githubMessages[locale];
    return extra ? { ...messages, github: extra } : messages;
};
