import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import { NativeStartup } from './app/control/NativeStartup'
import { lazy, Suspense } from 'react'
import { AppErrorBoundary } from './app/data/AppErrorBoundary'
// The marketing wall is a standalone page: /app/wall (the server's app fallback already serves it).
const Wall = lazy(() => import('./app/wall/WallMount'))
const isWall = /\/wall\/?$/.test(window.location.pathname)

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <AppErrorBoundary>
            {isWall ? <Suspense fallback={null}><Wall/></Suspense> : <NativeStartup/>}
        </AppErrorBoundary>
    </React.StrictMode>
)
