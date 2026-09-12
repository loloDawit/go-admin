import { FunctionComponent, useEffect, useState } from 'react';
import { Navigate } from 'react-router-dom';
import { api } from '../api/client';
import { UserInfo } from '../interfaces/user';
import Menu from './Menu';
import Nav from './Nav';

interface LayoutProps {}

const Layout: FunctionComponent<LayoutProps> = ({ children }) => {
  const [redirect, setRedirect] = useState(false);
  useEffect(() => {
    (async () => {
      try {
        await api<UserInfo>('/user');
      } catch (error) {
        setRedirect(true);
      }
    })();
  }, []);
  if (redirect) {
    return <Navigate to={'/login'} />;
  }
  return (
    <>
      <Nav />
      <div className="container-fluid">
        <div className="row">
          <Menu />
          <main className="col-md-9 ms-sm-auto col-lg-10 px-md-4">{children}</main>
        </div>
      </div>
    </>
  );
};

export default Layout;
