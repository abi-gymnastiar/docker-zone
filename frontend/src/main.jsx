import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { api } from './api'
import HomePage from './pages/HomePage'
import AdminPage from './pages/AdminPage'
import LoginPage from './pages/LoginPage'
import ServicePage from './pages/ServicePage'
import './style.css'

function App() {
  const serviceName = location.pathname.startsWith('/service/') ? location.pathname.split('/')[2] : null
  const isAdmin = location.pathname === '/admin'
  const [services, setServices] = useState([])
  const [service, setService] = useState(null)
  const [user, setUser] = useState(undefined)
  const [message, setMessage] = useState('')
  const refresh = () => api('/services').then(setServices).catch((error) => setMessage(error.message))

  useEffect(() => { api('/auth/me').then(setUser).catch(() => setUser(null)) }, [])
  useEffect(() => { if (user) refresh() }, [user])
  useEffect(() => {
    if (serviceName) api(`/services/${serviceName}`).then(setService).catch((error) => setMessage(error.message))
  }, [serviceName])

  if (user === undefined) return <div />
  if (!user) return <LoginPage onLogin={(loggedInUser) => { setUser(loggedInUser); refresh() }} />
  if (isAdmin && user.role === 'admin') return <AdminPage user={user} onLogout={() => api('/auth/logout', { method: 'POST' }).then(() => setUser(null))} message={message} setMessage={setMessage} />
  if (serviceName) return <ServicePage service={service} serviceName={serviceName} onRefresh={refresh}
    message={message} setMessage={setMessage} setService={setService} />
  const logout = () => api('/auth/logout', { method: 'POST' }).then(() => setUser(null))
  return <HomePage services={services} message={message} user={user} onLogout={logout} />
}

createRoot(document.getElementById('root')).render(<StrictMode><App /></StrictMode>)
