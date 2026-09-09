import { useState } from 'react'
import { api, getLogs } from '../api'
import ActionButtons from '../components/ActionButtons'
import Layout from '../components/Layout'
import Status from '../components/Status'
import './ServicePage.css'

export default function ServicePage({ service, serviceName, user, onRefresh, message, setMessage, setService }) {
  const [logs, setLogs] = useState('')
  const runAction = (action) => {
    setMessage(`Running ${action}...`)
    api(`/services/${serviceName}/action`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action }),
    }).then(() => {
      setMessage(`${action} complete.`)
      onRefresh()
      return api(`/services/${serviceName}`)
    }).then(setService).catch((error) => setMessage(error.message))
  }
  const loadLogs = () => getLogs(serviceName).then(setLogs).catch((error) => setMessage(error.message))

  return <Layout>
    <header><a href="/">◄ BACK TO CONTROL PANEL</a>{user?.role === 'admin' && <a className="admin-link" href={`/admin?service=${encodeURIComponent(serviceName)}`}>CONFIGURE CONTAINER</a>}<h1>{service?.name || serviceName}</h1></header>
    <section className="panel"><h2>CONTAINER STATUS</h2>
      {service ? <><Status running={service.running} /><p>{service.description}</p>
        <p>Docker ID: {service.containerId || 'not found'}</p>
        <ActionButtons actions={service.actions} running={service.running} onAction={runAction} onLogs={loadLogs} />
      </> : <p>Loading service...</p>}
    </section>
    {logs && <section className="panel"><h2>RECENT LOGS</h2><pre>{logs}</pre></section>}
    <p className="message">{message}</p>
  </Layout>
}
