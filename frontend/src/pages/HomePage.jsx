import Layout from '../components/Layout'
import ServiceCard from '../components/ServiceCard'

export default function HomePage({ services, message }) {
  return <Layout>
    <header><h1>★ MY DOCKER ZONE ★</h1><p>tiny control panel / very serious technology</p></header>
    <section className="panel"><h2>CONTAINERS</h2>
      {services.map((service) => <ServiceCard service={service} key={service.name} />)}
      {!services.length && <p>No configured services found.</p>}
    </section>
    <p className="message">{message}</p>
  </Layout>
}
