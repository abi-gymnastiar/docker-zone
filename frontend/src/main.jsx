import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { api } from './api'
import HomePage from './pages/HomePage'
import ServicePage from './pages/ServicePage'
import './style.css'

function App() {
  const serviceName = location.pathname.startsWith('/service/') ? location.pathname.split('/')[2] : null
  const [services, setServices] = useState([])
  const [service, setService] = useState(null)
  const [message, setMessage] = useState('')
  const refresh = () => api('/services').then(setServices).catch((error) => setMessage(error.message))

  useEffect(() => { refresh() }, [])
  useEffect(() => {
    if (serviceName) api(`/services/${serviceName}`).then(setService).catch((error) => setMessage(error.message))
  }, [serviceName])

  if (serviceName) return <ServicePage service={service} serviceName={serviceName} onRefresh={refresh}
    message={message} setMessage={setMessage} setService={setService} />
  return <HomePage services={services} message={message} />
}

createRoot(document.getElementById('root')).render(<StrictMode><App /></StrictMode>)
