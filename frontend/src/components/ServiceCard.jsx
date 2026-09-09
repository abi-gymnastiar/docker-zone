import Status from './Status'
import './ServiceCard.css'

export default function ServiceCard({ service }) {
  return <a className="service" href={`/service/${service.name}`}>
    <img src={service.iconUrl} alt="" /><span><strong>{service.name}</strong><small>{service.description}</small></span>
    <Status running={service.running} />
  </a>
}
