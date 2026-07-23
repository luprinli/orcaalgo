import { useTranslation } from 'react-i18next'

export default function BrokerManagement() {
  const { t } = useTranslation()
  return <div>
    <h1 style={{margin:0}}>{t('sidebar:nav.brokers', 'Brokers')}</h1>
    <div className="card mt-4"><p className="text-muted">{t('brokers:description', 'Broker adapters: Paper, Alpaca, IBKR. Configure broker credentials in Credentials section.')}</p></div>
  </div>
}
