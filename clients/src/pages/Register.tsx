import { FunctionComponent, SyntheticEvent, useState } from 'react';
import { Navigate } from 'react-router-dom';
import axios from 'axios';
import './Register.css';

interface RegisterProps {}

const Register: FunctionComponent<RegisterProps> = () => {
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [redirect, setRedirect] = useState(false);

  const submit = async (e: SyntheticEvent) => {
    e.preventDefault();
    let data = {
      firstName,
      lastName,
      email,
      password,
      passwordConfirm
    };
    var config = {
      method: 'post',
      url: 'http://localhost:8080/api/v1/register',
      headers: {
        'Content-Type': 'application/json'
      },
      data: data
    };
    //@ts-ignore
    const response = await axios(config);
    console.log(response);
    setRedirect(true);
  };
  return (
    <>
      {redirect ? (
        <Navigate to={'/login'} />
      ) : (
        <main className="form-signin">
          <form onSubmit={submit}>
            <h1 className="h3 mb-3 fw-normal">Please register</h1>

            <input
              className="form-control"
              placeholder="First Name"
              required
              onChange={(e) => setFirstName(e.target.value)}
              value={firstName}
            />

            <input
              className="form-control"
              placeholder="Last Name"
              required
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
            />

            <input
              type="email"
              className="form-control"
              placeholder="Email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />

            <input
              type="password"
              className="form-control"
              placeholder="Password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />

            <input
              type="password"
              className="form-control"
              placeholder="Password Confirm"
              required
              value={passwordConfirm}
              onChange={(e) => setPasswordConfirm(e.target.value)}
            />

            <button className="w-100 btn btn-lg btn-primary" type="submit">
              Submit
            </button>
          </form>
        </main>
      )}
    </>
  );
};

export default Register;
