import { createBrowserRouter } from 'react-router-dom'
import { AppShell } from './AppShell'
import { Dashboard } from '../routes/Dashboard'
import { Login } from '../routes/Login'
import { Orders } from '../routes/Orders'
import { OrderDetail } from '../routes/OrderDetail'
import { Products } from '../routes/Products'
import { ProductDetail } from '../routes/ProductDetail'
import { ProductForm } from '../routes/ProductForm'
import { Customers } from '../routes/Customers'
import { CustomerDetail } from '../routes/CustomerDetail'
import { Staff } from '../routes/Staff'
import { Roles } from '../routes/Roles'
import { Permissions } from '../routes/Permissions'
import { Profile } from '../routes/Profile'
import { Settings } from '../routes/Settings'
import { Kit } from '../routes/Kit'
import { Forbidden, NotFound, ServerError } from '../routes/Problem'

export const router = createBrowserRouter([
  { path: '/login', element: <Login /> },
  {
    element: <AppShell />,
    children: [
      { path: '/', element: <Dashboard /> },
      { path: '/orders', element: <Orders /> },
      { path: '/orders/:orderId', element: <OrderDetail /> },
      { path: '/products', element: <Products /> },
      { path: '/products/new', element: <ProductForm /> },
      { path: '/products/:productId', element: <ProductDetail /> },
      { path: '/products/:productId/edit', element: <ProductForm /> },
      { path: '/customers', element: <Customers /> },
      { path: '/customers/:customerId', element: <CustomerDetail /> },
      { path: '/staff', element: <Staff /> },
      { path: '/roles', element: <Roles /> },
      { path: '/permissions', element: <Permissions /> },
      { path: '/profile', element: <Profile /> },
      { path: '/settings', element: <Settings /> },
      { path: '/kit', element: <Kit /> },
      { path: '/403', element: <Forbidden /> },
      { path: '/500', element: <ServerError /> },
      { path: '*', element: <NotFound /> },
    ],
  },
])
