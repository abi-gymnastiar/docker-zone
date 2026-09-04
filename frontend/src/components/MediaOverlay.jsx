import { useMemo, useState } from 'react'
import './MediaOverlay.css'

const media = Object.values(import.meta.glob('../media/*.{gif,png,jpg,jpeg,webp,svg}', {
  eager: true,
  query: '?url',
  import: 'default',
}))

function shuffled(items) {
  return [...items].sort(() => Math.random() - 0.5).slice(0, 3)
}

export default function MediaOverlay() {
  const [open, setOpen] = useState(false)
  const selectedMedia = useMemo(() => shuffled(media), [])

  return <>
    <button className="evil-button" onClick={() => setOpen(!open)} aria-label="Toggle evil media overlay">
      {open ? 'HIDE EVIL' : 'EVIL BUTTON'}
    </button>
    {open && <div className="media-overlay" role="dialog" aria-label="Evil media overlay">
      <div className="media-corners">
        {selectedMedia.map((source, index) => <img className={`corner-${index}`} src={source} alt="" key={`${source}-${index}`} />)}
        {!selectedMedia.length && <p>Add pictures or GIFs to <code>frontend/src/media/</code> and rebuild.</p>}
      </div>
    </div>}
  </>
}
