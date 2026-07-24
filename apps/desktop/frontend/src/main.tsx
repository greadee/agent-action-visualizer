import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '@prool-ui/tokens/tokens.css'
import '@prool-ui/themes/themes.css'
import '@prool-ui/react/styles.css'
import './style.css'
import App from './App'

document.documentElement.dataset.puiTheme = 'dark'
const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
  <StrictMode>
    <App />
  </StrictMode>,
)
