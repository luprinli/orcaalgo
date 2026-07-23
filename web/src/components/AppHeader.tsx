import { useEffect } from 'react'
import nprogress from 'nprogress'
import 'nprogress/nprogress.css'
import { registerLoader } from '../api/middleware'

nprogress.configure({ showSpinner: false, speed: 300, trickleSpeed: 150 })

export default function AppHeader() {
  useEffect(() => {
    registerLoader({
      start: () => nprogress.start(),
      stop: () => nprogress.done(),
    })
  }, [])

  return null
}
