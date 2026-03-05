import React from 'react';
import ReactDOM from 'react-dom/client';
import { Provider } from 'react-redux';
import { store } from './store/store';
import { Web3Provider } from './providers/Web3Provider';
import { LanguageProvider } from './context/LanguageContext';
import App from './App';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Provider store={store}>
      <Web3Provider>
        <LanguageProvider>
          <App />
        </LanguageProvider>
      </Web3Provider>
    </Provider>
  </React.StrictMode>,
);
