import Layout from '../components/Layout'
import ServiceCard from '../components/ServiceCard'
import Pagination from '../components/Pagination'

export default function HomePage({ services, message, user, onLogout, page, pageSize, total, search, setSearch, onPageChange }) {
  return <Layout>
    <header><h1>★ MY DOCKER ZONE ★</h1><p>tiny control panel / very serious technology</p><p>logged in as {user.username} ({user.role}) {user.role === 'admin' && <a href="/admin">[ ADMIN MACHINE ]</a>} <button onClick={onLogout}>LOG OUT</button></p></header>
    <section className="panel"><h2>CONTAINERS</h2><input className="search" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="search services..." />
      {services.map((service) => <ServiceCard service={service} key={service.name} />)}
      {!services.length && <p>No configured services found.</p>}
      <Pagination page={page} pageSize={pageSize} total={total} onChange={onPageChange} />
    </section>
    <p className="message">{message}</p>
  </Layout>
}
