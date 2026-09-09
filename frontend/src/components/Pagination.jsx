import './Pagination.css'

export default function Pagination({ page, pageSize, total, onChange }) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  return <div className="pagination">
    <label>PER PAGE <select value={pageSize} onChange={(event) => onChange(1, Number(event.target.value))}>
      {[5, 10, 20, 50].map((size) => <option value={size} key={size}>{size}</option>)}
    </select></label>
    <button disabled={page <= 1} onClick={() => onChange(page - 1, pageSize)}>PREV</button>
    <span>PAGE {page} / {pages} ({total})</span>
    <button disabled={page >= pages} onClick={() => onChange(page + 1, pageSize)}>NEXT</button>
  </div>
}
