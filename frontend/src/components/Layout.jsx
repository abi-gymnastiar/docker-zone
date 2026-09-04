import './Layout.css'
import MediaOverlay from './MediaOverlay'

export default function Layout({ children }) {
  return <>
    <main>{children}</main>
    <MediaOverlay />
    <footer>docker zone, Developed by Jimi - with love &lt;3</footer>
  </>
}
