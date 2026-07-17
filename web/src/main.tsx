import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from '@/App'
import { AppProviders } from '@/app/providers'
import { AppErrorBoundary } from '@/app/app-error-boundary'
import '@/index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppErrorBoundary>
      <AppProviders>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </AppProviders>
    </AppErrorBoundary>
  </React.StrictMode>,
)
