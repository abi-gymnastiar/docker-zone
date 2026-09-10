import Layout, { useBackground } from '../components/Layout'
import ServiceCard from '../components/ServiceCard'
import Pagination from '../components/Pagination'

import { useEffect, useState } from 'react'
import { api } from '../api'
import './HomePage.css'

export default function HomePage({ services, message, user, onLogout, page, pageSize, total, search, setSearch, onPageChange }) {
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [config, setConfig] = useState(null)
  const [background, setBackground] = useBackground()
  const [favorites, setFavorites] = useState(() => JSON.parse(localStorage.getItem('dashboard-favorites') || '[]'))
  const [favoriteServices, setFavoriteServices] = useState([])
  useEffect(() => {
    api('/config').then(value => {
      setConfig(value)
      if (!localStorage.getItem('dashboard-background') && value.web?.backgroundImage) {
        setBackground({ ...background, images: [value.web.backgroundImage], scale: value.web.backgroundScale || 'tile' })
      }
    }).catch(() => {})
  }, [])
  useEffect(() => {
    const syncFavorites = () => setFavorites(JSON.parse(localStorage.getItem('dashboard-favorites') || '[]'))
    window.addEventListener('dashboard-favorites-change', syncFavorites)
    window.addEventListener('storage', syncFavorites)
    return () => {
      window.removeEventListener('dashboard-favorites-change', syncFavorites)
      window.removeEventListener('storage', syncFavorites)
    }
  }, [])
  useEffect(() => {
    let active = true
    Promise.all(favorites.map(name => api(`/services/${encodeURIComponent(name)}`)))
      .then(result => active && setFavoriteServices(result))
      .catch(() => active && setFavoriteServices([]))
    return () => { active = false }
  }, [favorites])
  const card = service => <ServiceCard service={service} key={service.name} />
  const updateImages = value => setBackground({ ...background, images: value.split(',').map(item => item.trim()).filter(Boolean) })
  const web = config?.web || {}
  return <Layout>
    <header><h1>{web.header || '★ MY DOCKER ZONE ★'}</h1><p>{web.subheader || 'tiny control panel / very serious technology'}</p><p>logged in as {user.username} ({user.role}) {user.role === 'admin' && <a href="/admin">[ ADMIN MACHINE ]</a>} <button onClick={onLogout}>LOG OUT</button> <button onClick={() => setSettingsOpen(true)}>BACKGROUND SETTINGS</button></p></header>
    <section className="panel">{favoriteServices.length > 0 && <><h2>FAVORITED CONTAINERS</h2>{favoriteServices.map(card)}</>}<h2>CONTAINERS</h2><input className="search" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="search services..." />
      {services.map(card)}
      {!services.length && <p>No configured services found.</p>}
      <Pagination page={page} pageSize={pageSize} total={total} onChange={onPageChange} />
    </section>
    <p className="message">{message}</p>
    {settingsOpen && <div className="settings-modal"><div className="settings-box"><h2>BACKGROUND SETTINGS</h2><label>Color <input type="color" value={background.color} onChange={event => setBackground({ ...background, color: event.target.value })} /></label><label>Image URLs (comma separated) <input value={background.images.join(', ')} onChange={event => updateImages(event.target.value)} /></label><label>Image scaling <select value={background.scale || 'tile'} onChange={event => setBackground({ ...background, scale: event.target.value })}><option value="tile">Tile</option><option value="stretch">Stretch</option><option value="zoom">Zoom to fill</option></select></label><button onClick={() => setSettingsOpen(false)}>CLOSE</button></div></div>}
  </Layout>
}
