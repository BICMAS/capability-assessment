export default function ReviewLoading() {
  return (
    <div className="role-profile" aria-busy="true" aria-live="polite">
      <p className="empty-state">Loading review and approvals…</p>
    </div>
  );
}