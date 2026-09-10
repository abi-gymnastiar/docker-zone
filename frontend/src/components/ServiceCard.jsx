import Status from './Status'
import './ServiceCard.css'

export default function ServiceCard({ service, favorite, onToggleFavorite }) {
  return <div className="service">
    <a className="service-link" href={`/service/${service.name}`}>
      <img src={service.iconUrl} alt="" /><span><strong>{service.name}</strong><small>{service.description}</small></span>
    </a>
    <Status running={service.running} />
  </div>
}
