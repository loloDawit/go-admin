import { FunctionComponent, SyntheticEvent, useState } from 'react';
import { Navigate } from 'react-router';
import axios from 'axios';

interface LoginProps {}

const Login: FunctionComponent<LoginProps> = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [redirect, setRedirect] = useState(false);

  const submit = async (e: SyntheticEvent) => {
    e.preventDefault();
    let data = {
      email,
      password
    };

    var config = {
      method: 'post',
      url: 'http://localhost:8080/api/v1/login',
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      },
      withCredentials: true,
      data: data
    };
    //@ts-ignore
    const { data: we } = await axios(config);
    console.log(we);

    setRedirect(true);
  };

  if (redirect) {
    return <Navigate to={'/'} />;
  }

  return (
    <main className="form-signin">
      <form onSubmit={submit}>
        <h1 className="h3 mb-3 fw-normal">Please sign in</h1>

        <input
          type="email"
          className="form-control"
          placeholder="Email"
          required
          onChange={(e) => setEmail(e.target.value)}
        />

        <input
          type="password"
          className="form-control"
          placeholder="Password"
          required
          onChange={(e) => setPassword(e.target.value)}
        />

        <button className="w-100 btn btn-lg btn-primary" type="submit">
          Submit
        </button>
      </form>
    </main>
  );
};

export default Login;
