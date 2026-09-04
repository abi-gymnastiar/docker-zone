import './ActionButtons.css'

export default function ActionButtons({ actions, running, onAction, onLogs }) {
  return <div className="buttons">
    {actions?.map((action) => <button key={action}
      disabled={(action === 'start' && running) || (action === 'stop' && !running)}
      onClick={() => onAction(action)}>{action.toUpperCase()}</button>)}
    <button onClick={onLogs}>VIEW LOGS</button>
  </div>
}
