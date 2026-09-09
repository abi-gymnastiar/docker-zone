import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { api, getPage } from './api'
import HomePage from './pages/HomePage'
import AdminPage from './pages/AdminPage'
import LoginPage from './pages/LoginPage'
import ServicePage from './pages/ServicePage'
import './style.css'

function App() {
  const serviceName = location.pathname.startsWith('/service/') ? location.pathname.split('/')[2] : null
  const isAdmin = location.pathname === '/admin'
  const [services, setServices] = useState([])
  const [serviceList, setServiceList] = useState({ page: 1, pageSize: 5, total: 0 })
  const [search, setSearch] = useState('')
  const [service, setService] = useState(null)
  const [user, setUser] = useState(undefined)
  const [message, setMessage] = useState('')
  const refresh = (page = serviceList.page, pageSize = serviceList.pageSize, query = search) =>
    getPage('/services', { page, pageSize, search: query }).then((result) => {
      setServices(result.items || [])
      setServiceList(result)
    }).catch((error) => setMessage(error.message))

  useEffect(() => { api('/auth/me').then(setUser).catch(() => setUser(null)) }, [])
  useEffect(() => { if (user) refresh(1, serviceList.pageSize, search) }, [user, search])
  useEffect(() => {
    if (serviceName) api(`/services/${serviceName}`).then(setService).catch((error) => setMessage(error.message))
  }, [serviceName])

  if (user === undefined) return <div />
  if (!user) return <LoginPage onLogin={(loggedInUser) => { setUser(loggedInUser); refresh() }} />
  if (isAdmin && user.role === 'admin') return <AdminPage user={user} onLogout={() => api('/auth/logout', { method: 'POST' }).then(() => setUser(null))} message={message} setMessage={setMessage} />
  if (serviceName) return <ServicePage service={service} serviceName={serviceName} onRefresh={refresh}
    message={message} setMessage={setMessage} setService={setService} />
  const logout = () => api('/auth/logout', { method: 'POST' }).then(() => setUser(null))
  return <HomePage services={services} message={message} user={user} onLogout={logout}
    page={serviceList.page} pageSize={serviceList.pageSize} total={serviceList.total}
    search={search} setSearch={setSearch} onPageChange={(page, pageSize) => refresh(page, pageSize)} />
}

createRoot(document.getElementById('root')).render(<StrictMode><App /></StrictMode>)
