export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const params = await searchParams;

  return (
    <main className="entrance">
      <div className="entrance-title">
        <p className="mark">BICMAS</p>
        <h1>Learn</h1>
        <p className="entrance-line">AI coaches. Managers develop. People perform.</p>
      </div>
      <section className="entrance-card">
        {params.error ? <p className="login-error">Sign-in failed.</p> : null}
        <form className="entrance-choices" action="/session" method="post">
          <button type="submit" name="person" value="ada">
            <span className="entrance-name">Ada</span>
            <span className="entrance-role">Framework admin</span>
          </button>
          <button type="submit" name="person" value="eli">
            <span className="entrance-name">Eli</span>
            <span className="entrance-role">Employee</span>
          </button>
          <button type="submit" name="person" value="maya">
            <span className="entrance-name">Maya</span>
            <span className="entrance-role">Line manager</span>
          </button>
        </form>
      </section>
    </main>
  );
}
