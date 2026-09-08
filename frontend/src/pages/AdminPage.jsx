import { useEffect, useState } from 'react'
import { api } from '../api'
import Layout from '../components/Layout'
import './AdminPage.css'

const empty = { users: [], groups: [], services: [] }

export default function AdminPage({ user, onLogout, message, setMessage }) {
  const [data, setData] = useState(empty)
  const [newUser, setNewUser] = useState({ username: '', password: '', role: 'viewer' })
  const [newGroup, setNewGroup] = useState('')
  const [membership, setMembership] = useState({ groupId: '', userId: '', role: 'viewer' })

  const refresh = () => Promise.all([
    api('/admin/users'),
    api('/admin/groups'),
    api('/admin/services'),
  ]).then(([users, groups, services]) => setData({ users, groups, services }))
    .catch((error) => setMessage(error.message))
  useEffect(() => { refresh() }, [])

  const submitUser = (event) => {
    event.preventDefault()
    api('/admin/users', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(newUser) })
      .then(() => { setMessage('User created.'); setNewUser({ username: '', password: '', role: 'viewer' }); return refresh() })
      .catch((error) => setMessage(error.message))
  }
  const submitGroup = (event) => {
    event.preventDefault()
    api('/admin/groups', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: newGroup }) })
      .then(() => { setMessage('Group created.'); setNewGroup(''); return refresh() })
      .catch((error) => setMessage(error.message))
  }
  const submitMembership = (event) => {
    event.preventDefault()
    api('/admin/groups', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ...membership, groupId: Number(membership.groupId), userId: Number(membership.userId) }) })
      .then(() => { setMessage('Membership saved.'); return refresh() })
      .catch((error) => setMessage(error.message))
  }
  const saveServiceGroups = (service, value) => {
    const groups = value.split(',').map((group) => group.trim()).filter(Boolean)
    api('/admin/services', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: service.name, groups }) })
      .then(() => { setMessage('Service groups saved.'); return refresh() })
      .catch((error) => setMessage(error.message))
  }

  return <Layout>
    <header><a href="/">◄ BACK TO CONTROL PANEL</a><h1>★ ADMIN MACHINE ★</h1><p>logged in as {user.username} <button onClick={onLogout}>LOG OUT</button></p></header>
    <section className="panel admin-grid">
      <div><h2>CREATE USER</h2><form onSubmit={submitUser}>
        <input placeholder="username" value={newUser.username} onChange={(event) => setNewUser({ ...newUser, username: event.target.value })} required />
        <input placeholder="password" type="password" value={newUser.password} onChange={(event) => setNewUser({ ...newUser, password: event.target.value })} required />
        <select value={newUser.role} onChange={(event) => setNewUser({ ...newUser, role: event.target.value })}><option value="viewer">viewer</option><option value="admin">admin</option></select>
        <button>MAKE USER</button>
      </form></div>
      <div><h2>CREATE GROUP</h2><form onSubmit={submitGroup}><input placeholder="group name" value={newGroup} onChange={(event) => setNewGroup(event.target.value)} required /><button>MAKE GROUP</button></form></div>
    </section>
    <section className="panel"><h2>GROUP MEMBERSHIP</h2><form className="inline-form" onSubmit={submitMembership}>
      <select value={membership.groupId} onChange={(event) => setMembership({ ...membership, groupId: event.target.value })} required><option value="">choose group</option>{data.groups.map((group) => <option value={group.id} key={group.id}>{group.name}</option>)}</select>
      <select value={membership.userId} onChange={(event) => setMembership({ ...membership, userId: event.target.value })} required><option value="">choose user</option>{data.users.map((item) => <option value={item.id} key={item.id}>{item.username}</option>)}</select>
      <select value={membership.role} onChange={(event) => setMembership({ ...membership, role: event.target.value })}><option>viewer</option><option>log_viewer</option><option>operator</option></select>
      <button>SAVE MEMBER</button>
    </form>{data.groups.map((group) => <p key={group.id}><b>{group.name}:</b> {group.members?.map((member) => `${member.username} (${member.role})`).join(', ') || 'empty'}</p>)}</section>
    <section className="panel"><h2>SERVICE GROUP ASSIGNMENTS</h2>{data.services.map((service) => <div className="service-assignment" key={service.name}><b>{service.name}</b><input defaultValue={service.groups.join(', ')} placeholder="group-one, group-two" onBlur={(event) => saveServiceGroups(service, event.target.value)} /></div>)}</section>
    <p className="message">{message}</p>
  </Layout>
}
