import { FunctionComponent, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api/client';
import { UserInfo } from '../interfaces/user';

interface NavProps {}

const Nav: FunctionComponent<NavProps> = () => {
  const [user, setUser] = useState<UserInfo | null>(null);

  const logout = async () => {
    try {
      await api('/logout', { method: 'POST' });
    } catch (error) {
      // The Link's own navigation to /login proceeds regardless of this result.
      console.error('logout request failed; session cookie may still be set', error);
    }
  };

  useEffect(() => {
    (async () => {
      try {
        const data = await api<UserInfo>('/user');
        setUser(data);
      } catch (error) {
        setUser(null);
      }
    })();
  }, []);

  return (
    <nav className="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow">
      <Link className="navbar-brand col-md-3 col-lg-2 mr-0 px-3" to="/">
        Go React Admin
      </Link>

      <ul className="navbar-nav flex-row my-2 my-md-0 mr-md-3">
        <li>
          <Link to="/profile" className="p-2 text-white text-decoration-none">
            {user?.firstName}
          </Link>
        </li>
        <li>
          <Link to="/login" className="p-2 text-white text-decoration-none" onClick={logout}>
            Sign out
          </Link>
        </li>
      </ul>
    </nav>
  );
};

export default Nav;
