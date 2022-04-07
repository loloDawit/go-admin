import axios from 'axios';
import { FunctionComponent, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { UserInfo } from '../interfaces/user';

interface NavProps {}

const Nav: FunctionComponent<NavProps> = () => {
  const [user, setUser] = useState<UserInfo | null>(null);

  const logout = async () => {
    var config = {
      method: 'post',
      url: 'http://localhost:8080/api/v1/logout',
      headers: {
        'Content-Type': 'application/json'
      },
      withCredentials: true
    };
    //@ts-ignore
    await axios(config);
  };
  useEffect(() => {
    (async () => {
      var config = {
        method: 'get',
        url: 'http://localhost:8080/api/v1/user',
        headers: {
          'Content-Type': 'application/json'
        },
        withCredentials: true
      };
      //@ts-ignore
      const { data } = await axios(config);
      setUser(data);
    })();
  }, []);

  return (
    <nav className="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow">
      <a className="navbar-brand col-md-3 col-lg-2 mr-0 px-3" href="#">
        Go React Admin
      </a>

      <ul className="my-2 my-md-0 mr-md-3">
        <Link to="/profile" className="p-2 text-white text-decoration-none">
          {user?.firstName}
        </Link>
        <Link to="/login" className="p-2 text-white text-decoration-none" onClick={logout}>
          Sign out
        </Link>
      </ul>
    </nav>
  );
};

export default Nav;
