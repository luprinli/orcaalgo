import { useTranslation } from 'react-i18next'

export default function LLMSettings() {
  const { t } = useTranslation()
  return <div>
    <h1 style={{margin:0}}>{t('sidebar:nav.llm', 'LLM Settings')}</h1>
    <div className="card mt-4"><p className="text-muted">{t('llm:description', 'Configure LLM provider (OpenAI / Anthropic / Ollama) for strategy analysis and trade commentary.')}</p></div>
  </div>
}
