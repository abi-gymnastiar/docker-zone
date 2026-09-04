import Status from './Status'
import './ServiceCard.css'

export default function ServiceCard({ service }) {
  return <a className="service" href={`/service/${service.name}`}>
    <strong>{service.name}</strong>
    <Status running={service.running} />
    <small>{service.description}</small>
  </a>
}
