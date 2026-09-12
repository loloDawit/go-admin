import { FunctionComponent, SyntheticEvent, useState } from 'react';
import { Navigate } from 'react-router';
import { api, ApiRequestError } from '../api/client';

interface LoginProps {}

const Login: FunctionComponent<LoginProps> = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [redirect, setRedirect] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const submit = async (e: SyntheticEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api('/login', { method: 'POST', body: JSON.stringify({ email, password }) });
      setRedirect(true);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : 'Something went wrong. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  if (redirect) {
    return <Navigate to={'/'} />;
  }

  return (
    <main className="form-signin">
      <form onSubmit={submit}>
        <h1 className="h3 mb-3 fw-normal">Please sign in</h1>

        {error && (
          <div className="alert alert-danger" role="alert">
            {error}
          </div>
        )}

        <label htmlFor="login-email" className="visually-hidden">
          Email
        </label>
        <input
          id="login-email"
          type="email"
          className="form-control"
          placeholder="Email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />

        <label htmlFor="login-password" className="visually-hidden">
          Password
        </label>
        <input
          id="login-password"
          type="password"
          className="form-control"
          placeholder="Password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        <button className="w-100 btn btn-lg btn-primary" type="submit" disabled={submitting}>
          Submit
        </button>

        <p className="mt-3 text-muted">
          Accounts are created by an administrator. Contact yours to request access.
        </p>
      </form>
    </main>
  );
};

export default Login;
