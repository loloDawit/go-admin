import { FunctionComponent } from 'react';
import Menu from './Menu';
import Nav from './Nav';

interface LayoutProps {}

const Layout: FunctionComponent<LayoutProps> = ({ children }) => {
  const [redirect, setRedirect] = useState(false);
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

      try {
        //@ts-ignore
        await axios(config);
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
