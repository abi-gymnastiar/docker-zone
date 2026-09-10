import { useState } from 'react'
import { api } from '../api'
import Layout from '../components/Layout'
import './LoginPage.css'

export default function LoginPage({ onLogin }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')

  const submit = (event) => {
    event.preventDefault()
    setMessage('Checking credentials...')
    api('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    }).then(onLogin).catch((error) => setMessage(error.message))
  }

  return <Layout>
    <section className="panel login-panel">
      <h1>★ DOCKER ZONE LOGIN ★</h1>
      <p>authorized weirdos only</p>
      <form onSubmit={submit}>
        <label>USERNAME<input value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" required /></label>
        <label>PASSWORD<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="current-password" required /></label>
        <button type="submit">ENTER THE ZONE</button>
      </form>
      <p className="message">{message}</p>
    </section>
  </Layout>
}
