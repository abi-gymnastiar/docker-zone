import './Status.css'

export default function Status({ running }) {
  return <span className={running ? 'status up' : 'status down'}>{running ? '● ONLINE' : '● OFFLINE'}</span>
}
