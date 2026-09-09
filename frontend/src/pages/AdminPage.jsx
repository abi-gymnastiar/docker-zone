import { useEffect, useState } from 'react'
import { api, getPage } from '../api'
import Layout from '../components/Layout'
import Pagination from '../components/Pagination'
import './AdminPage.css'

const empty = { users: [], groups: [], services: [] }

export default function AdminPage({ user, onLogout, message, setMessage }) {
  const [data, setData] = useState(empty)
  const [newUser, setNewUser] = useState({ username: '', password: '', role: 'viewer' })
  const [newGroup, setNewGroup] = useState('')
  const [membership, setMembership] = useState({ groupId: '', userId: '', role: 'viewer' })
  const [serviceList, setServiceList] = useState({ page: 1, pageSize: 5, total: 0 })
  const [search, setSearch] = useState('')
  const [editingService, setEditingService] = useState(null)

  const refresh = () => Promise.all([
    api('/admin/users'),
    api('/admin/groups'),
    getPage('/admin/services', { page: serviceList.page, pageSize: serviceList.pageSize, search }),
  ]).then(([users, groups, serviceResult]) => setData({
    users: users || [],
    groups: groups || [],
    services: (serviceResult.items || []).map((service) => ({
      ...service,
      groups: service.groups || [],
      actions: service.actions || [],
    })),
  })); setServiceList(serviceResult)
    .catch((error) => setMessage(error.message))
  useEffect(() => { refresh() }, [search, serviceList.page, serviceList.pageSize])

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
    api('/admin/services', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: service.name, groups, description: service.description, enabled: service.enabled, actions: service.actions }) })
      .then(() => { setMessage('Service groups saved.'); return refresh() })
      .catch((error) => setMessage(error.message))
  }
  const saveService = (service, changes) => api('/admin/services', {
    method: 'PUT', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...service, ...changes }),
  }).then(() => { setMessage('Service settings saved.'); return refresh() }).catch((error) => setMessage(error.message))
  const syncServices = () => api('/admin/sync', { method: 'POST' })
    .then(() => { setMessage('Docker services synchronized.'); return refresh() })
    .catch((error) => setMessage(error.message))

  const saveEditingService = () => saveService(editingService, {
    groups: editingService.groups.split(',').map((group) => group.trim()).filter(Boolean),
  }).then(() => setEditingService(null))

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
    <section className="panel"><h2>SERVICE CATALOG</h2><button onClick={syncServices}>SYNC DOCKER SERVICES</button><input className="search" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="search discovered containers..." />{data.services.map((service) => <div className="service-assignment" key={service.name}>
      <b>{service.name}</b><small>{service.container} {service.orphaned && '(ORPHANED)'}</small><button onClick={() => setEditingService({ ...service, groups: (service.groups || []).join(', ') })}>CONFIGURE CONTAINER</button>
    </div>)}<Pagination page={serviceList.page} pageSize={serviceList.pageSize} total={serviceList.total} onChange={(page, pageSize) => setServiceList({ ...serviceList, page, pageSize })} /></section>
    {editingService && <div className="admin-modal"><div className="admin-modal-box"><h2>CONFIGURE {editingService.name}</h2>
      <input value={editingService.description} placeholder="description" onChange={(event) => setEditingService({ ...editingService, description: event.target.value })} />
      <input value={editingService.groups} placeholder="group-one, group-two" onChange={(event) => setEditingService({ ...editingService, groups: event.target.value })} />
      <label><input type="checkbox" checked={editingService.enabled} onChange={(event) => setEditingService({ ...editingService, enabled: event.target.checked })} /> ENABLE FOR USERS</label>
      <input value={(editingService.actions || []).join(', ')} placeholder="start, stop, restart" onChange={(event) => setEditingService({ ...editingService, actions: event.target.value.split(',').map((item) => item.trim()).filter(Boolean) })} />
      <button onClick={saveEditingService}>SAVE</button><button onClick={() => setEditingService(null)}>CANCEL</button>
    </div></div>}
    <p className="message">{message}</p>
  </Layout>
}
