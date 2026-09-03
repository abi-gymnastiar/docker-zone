import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './style.css'

const api = (path, options) => fetch(`/api${path}`, options).then(async response => {
  if (!response.ok) throw new Error(await response.text() || response.statusText)
  return response.status === 204 ? null : response.json()
})

function Status({ running }) {
  return <span className={running ? 'status up' : 'status down'}>{running ? '● ONLINE' : '● OFFLINE'}</span>
}

function App() {
  const serviceName = location.pathname.startsWith('/service/') ? location.pathname.split('/')[2] : null
  const [services, setServices] = useState([])
  const [service, setService] = useState(null)
  const [logs, setLogs] = useState('')
  const [message, setMessage] = useState('')

  const refresh = () => api('/services').then(setServices).catch(error => setMessage(error.message))
  useEffect(() => { refresh() }, [])
  useEffect(() => {
    if (serviceName) api(`/services/${serviceName}`).then(setService).catch(error => setMessage(error.message))
  }, [serviceName])

  const runAction = action => {
    setMessage(`Running ${action}...`)
    api(`/services/${serviceName}/action`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action }) })
      .then(() => { setMessage(`${action} complete.`); refresh(); return api(`/services/${serviceName}`) }).then(setService).catch(error => setMessage(error.message))
  }
  const loadLogs = () => fetch(`/api/services/${serviceName}/logs`).then(response => response.text()).then(setLogs).catch(error => setMessage(error.message))

  if (serviceName) return <main><header><a href="/">◄ BACK TO CONTROL PANEL</a><h1>{service?.name || serviceName}</h1></header>
    <section className="panel"><h2>CONTAINER STATUS</h2>{service ? <><Status running={service.running} /><p>{service.description}</p><p>Docker ID: {service.containerId || 'not found'}</p><div className="buttons">
      {service.actions?.map(action => <button key={action} disabled={(action === 'start' && service.running) || (action === 'stop' && !service.running)} onClick={() => runAction(action)}>{action.toUpperCase()}</button>)}
      <button onClick={loadLogs}>VIEW LOGS</button>
    </div></> : <p>Loading service...</p>}</section>
    {logs && <section className="panel"><h2>RECENT LOGS</h2><pre>{logs}</pre></section>}<p className="message">{message}</p></main>

  return <main><header><h1>★ MY DOCKER ZONE ★</h1><p>tiny control panel / very serious technology</p></header><section className="panel"><h2>CONTAINERS</h2>{services.map(item => <a className="service" href={`/service/${item.name}`} key={item.name}><strong>{item.name}</strong><Status running={item.running} /><small>{item.description}</small></a>)}{!services.length && <p>No configured services found.</p>}</section><p className="message">{message}</p></main>
}

createRoot(document.getElementById('root')).render(<StrictMode><App /></StrictMode>)
