export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const params = await searchParams;

  return (
    <main className="entrance">
      <section className="entrance-frame">
        <div className="entrance-panel">
          <p className="mark">BICMAS</p>
          <h1>Learn</h1>
          <p className="entrance-line">AI coaches. Managers develop. People perform.</p>
        </div>
        <section className="entrance-card">
          <p className="mark">BICMAS</p>
          <h2>Log in</h2>
          {params.error ? <p className="login-error">Sign-in failed.</p> : null}
          <form className="entrance-form" action="/session" method="post" noValidate>
            <label htmlFor="email">Email address</label>
            <input id="email" name="email" type="email" autoComplete="username" />
            <label htmlFor="password">Password</label>
            <input id="password" name="password" type="password" autoComplete="current-password" />
            <button className="entrance-submit" type="submit">
              Log in
            </button>
          </form>
          <p className="entrance-demo">New here? Use a demo account.</p>
          <ul className="entrance-accounts">
            <li>ada@bicmas.learn · Framework admin</li>
            <li>eli@bicmas.learn · Employee</li>
            <li>maya@bicmas.learn · Line manager</li>
          </ul>
        </section>
      </section>
    </main>
  );
}
