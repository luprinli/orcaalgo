import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import en from './locales/en/translation.json'

const resources: Record<string, { translation: typeof en }> = {
  'en': { translation: en },
  'en-US': { translation: en },
}
// Register all top-level keys as namespaces so `dashboard:balance` resolves.
// i18next v24+ requires explicit namespace registration for `:` separator.
for (const [lang, langRes] of Object.entries(resources)) {
  const r = langRes as Record<string, unknown>
  for (const ns of Object.keys(en as Record<string, unknown>)) {
    if (!r[ns]) r[ns] = (en as Record<string, unknown>)[ns]
  }
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: resources as any,
    fallbackLng: 'en',
    debug: import.meta.env.DEV,
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      lookupLocalStorage: 'orca_lang',
    },
  })

export default i18n
