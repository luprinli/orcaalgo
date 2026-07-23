import { useState } from 'react'

export default function NotificationSettings() {
  const [email, setEmail] = useState(true)
  const [push, setPush] = useState(true)
  const [telegram, setTelegram] = useState(false)
  return <div>
    <h1 style={{margin:0}}>Notification Settings</h1>
    <div className="card mt-4"><h2>Channels</h2>
      <div style={{display:'flex',flexDirection:'column',gap:10}}>
        {[{k:'email',l:'Email Alerts',v:email,s:setEmail},{k:'push',l:'Push Notifications',v:push,s:setPush},{k:'telegram',l:'Telegram Alerts',v:telegram,s:setTelegram}].map(n=>
          <label key={n.k} className="flex gap-2" style={{alignItems:'center'}}><input type="checkbox" checked={n.v} onChange={e=>n.s(e.target.checked)}/> {n.l}</label>
        )}
      </div>
    </div>
  </div>
}
