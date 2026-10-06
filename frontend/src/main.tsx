import React, { lazy, Suspense } from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

const Specimen = import.meta.env.DEV && location.search.includes('specimen')
    ? lazy(() => import('./app/specimen/Specimen')) : undefined

root.render(
    <React.StrictMode>
        {Specimen ? <Suspense fallback={<p>Loading specimen…</p>}><Specimen/></Suspense> : <App/>}
    </React.StrictMode>
)
