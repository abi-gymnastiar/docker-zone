import './Layout.css'
import MediaOverlay from './MediaOverlay'
import { useEffect, useState } from 'react'
import { api } from '../api'

export default function Layout({ children }) {
  const [config, setConfig] = useState(null)
  useEffect(() => {
    let active = true
    const load = () => api('/config').then(value => active && setConfig(value)).catch(() => {})
    load()
    const timer = setInterval(load, 30000)
    return () => { active = false; clearInterval(timer) }
  }, [])
  useEffect(() => {
    if (config?.web?.title) document.title = config.web.title
    if (config?.web?.icon) {
      let link = document.querySelector('link[rel="icon"]')
      if (!link) {
        link = document.createElement('link')
        link.rel = 'icon'
        document.head.appendChild(link)
      }
      link.href = config.web.icon
    }
  }, [config])
  return <>
    <main>{children}</main>
    <MediaOverlay config={config} />
    <footer>{config?.web?.footer || 'docker zone, Developed by Jimi - with love <3'}</footer>
  </>
}
