import Status from './Status'
import './ServiceCard.css'

export default function ServiceCard({ service, favorite, onToggleFavorite }) {
  return <div className="service">
    <a className="service-link" href={`/service/${service.name}`}>
      <img src={service.iconUrl} alt="" /><span><strong>{service.name}</strong><small>{service.description}</small></span>
    </a>
    <Status running={service.running} />
    <button className="favorite-button" onClick={() => onToggleFavorite(service.name)} aria-label={favorite ? `Remove ${service.name} from favorites` : `Favorite ${service.name}`}>
      {favorite ? '★' : '☆'}
    </button>
  </div>
}
