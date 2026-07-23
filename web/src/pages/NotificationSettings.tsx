import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export default function NotificationSettings() {
  const { t } = useTranslation()
  const [email, setEmail] = useState(true)
  const [push, setPush] = useState(true)
  const [telegram, setTelegram] = useState(false)
  return <div>
    <h1 style={{margin:0}}>{t('sidebar:nav.notifications', 'Notification Settings')}</h1>
    <div className="card mt-4"><h2>{t('notification:channels', 'Channels')}</h2>
      <div style={{display:'flex',flexDirection:'column',gap:10}}>
        {[{k:'email',l:t('notification:emailAlerts', 'Email Alerts'),v:email,s:setEmail},{k:'push',l:t('notification:pushNotifications', 'Push Notifications'),v:push,s:setPush},{k:'telegram',l:t('notification:telegramAlerts', 'Telegram Alerts'),v:telegram,s:setTelegram}].map(n=>
          <label key={n.k} className="flex gap-2" style={{alignItems:'center'}}><input type="checkbox" checked={n.v} onChange={e=>n.s(e.target.checked)}/> {n.l}</label>
        )}
      </div>
    </div>
  </div>
}
