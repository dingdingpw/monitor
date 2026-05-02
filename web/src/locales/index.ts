import {createI18n} from "vue-i18n";
import en from './en.json';
import zh from './zh.json';
import ja from './ja.json';
import de from './de.json';
import ko from './ko.json';

const i18n = createI18n({
    legacy: false,
    locale: 'zh',
    locales: ['en', 'zh', 'ja', 'de', 'ko'],
    detectBrowserLanguage: false,
    messages: {
        en,
        zh,
        ja,
        de,
        ko
    }
});

export default i18n;