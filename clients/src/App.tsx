import React from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import Menu from './Components/Menu';
import Nav from './Components/Nav';
import Dashboard from './pages/Dashboards';
import Users from './pages/Users';
import './App.css';
import Register from './pages/Register';

interface AppProps {}

const App: React.FunctionComponent<AppProps> = () => {
  return (
    <div className="App">
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/users" element={<Users />} />
          <Route path="/register" element={<Register />} />
        </Routes>
      </BrowserRouter>
    </div>
  );
};

export default App;
