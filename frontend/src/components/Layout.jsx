import './Layout.css'
import MediaOverlay from './MediaOverlay'
import { useEffect, useState } from 'react'
import { api } from '../api'

export function useBackground() {
  const [background, setBackgroundState] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem('dashboard-background')) || { color: '#008080', images: [], scale: 'tile' }
    } catch {
      return { color: '#008080', images: [], scale: 'tile' }
    }
  })
  const setBackground = value => {
    setBackgroundState(value)
    localStorage.setItem('dashboard-background', JSON.stringify(value))
    window.dispatchEvent(new Event('dashboard-background-change'))
  }
  useEffect(() => {
    const sync = () => {
      try {
        setBackgroundState(JSON.parse(localStorage.getItem('dashboard-background')) || { color: '#008080', images: [], scale: 'tile' })
      } catch {
        // Keep the last valid preference when storage contains invalid JSON.
      }
    }
    window.addEventListener('dashboard-background-change', sync)
    window.addEventListener('storage', sync)
    return () => {
      window.removeEventListener('dashboard-background-change', sync)
      window.removeEventListener('storage', sync)
    }
  }, [])
  useEffect(() => {
    document.body.style.backgroundColor = background.color || '#008080'
    document.body.style.backgroundImage = background.images?.length
      ? `url("${background.images.join('"), url("')}")`
      : ''
    document.body.style.backgroundRepeat = background.scale === 'tile' ? 'repeat' : 'no-repeat'
    document.body.style.backgroundAttachment = 'fixed'
    document.body.style.backgroundSize = background.scale === 'stretch' ? '100vw 100vh' : background.scale === 'zoom' ? 'cover' : 'auto'
  }, [background])
  return [background, setBackground]
}

export default function Layout({ children }) {
  useBackground()
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
