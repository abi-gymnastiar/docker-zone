import { useEffect, useMemo, useState } from 'react'
import './MediaOverlay.css'

function shuffled(items) {
  return [...items].sort(() => Math.random() - 0.5).slice(0, 3)
}

function mediaURL(source) {
  return source.startsWith('/data/') ? `/api/config/media?path=${encodeURIComponent(source)}` : source
}

export default function MediaOverlay({ config }) {
  const [open, setOpen] = useState(() => localStorage.getItem('evil-overlay-open') === 'true')
  const media = config?.evil?.images || []
  const selectedMedia = useMemo(() => shuffled(media).map(mediaURL), [media])
  useEffect(() => {
    localStorage.setItem('evil-overlay-open', String(open))
  }, [open])

  if (!config?.evil) return null

  return <>
    <button className="evil-button" onClick={() => setOpen(!open)} aria-label="Toggle evil media overlay">
      {open ? 'HIDE EVIL' : 'EVIL BUTTON'}
    </button>
    {open && <div className="media-overlay" role="dialog" aria-label="Evil media overlay">
      <div className="media-corners">
        {selectedMedia.map((source, index) => <img className={`corner-${index}`} src={source} alt="" key={`${source}-${index}`} />)}
        {!selectedMedia.length && <p>Add image paths or URLs to the evil section in config.yml.</p>}
      </div>
    </div>}
  </>
}
