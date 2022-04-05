import { FunctionComponent } from 'react';
import { Link } from 'react-router-dom';

interface MenuProps {}

const Menu: FunctionComponent<MenuProps> = () => {
  return (
    <nav id="sidebarMenu" className="col-md-3 col-lg-2 d-md-block bg-light sidebar collapse">
      <div className="position-sticky pt-3">
        <ul className="nav flex-column">
          <li className="nav-item">
            <Link to={'/'} className="nav-link ">
              Dashboard
            </Link>
          </li>
          <li className="nav-item">
            <Link to={'/users'} className="nav-link ">
              Users
            </Link>
          </li>
        </ul>
      </div>
    </nav>
  );
};

export default Menu;
