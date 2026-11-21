import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import { NativeStartup } from './app/control/NativeStartup'
import { lazy, Suspense } from 'react'
import { plainBindings } from './app/data/plainFailure'

// Before any screen calls Go: an offline failure reads as one plain sentence everywhere.
plainBindings((window as { go?: unknown }).go)

// The marketing wall is a standalone page: /app/wall (the server's app fallback already serves it).
const Wall = lazy(() => import('./app/wall/WallMount'))
const isWall = /\/wall\/?$/.test(window.location.pathname)

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        {isWall ? <Suspense fallback={null}><Wall/></Suspense> : <NativeStartup/>}
    </React.StrictMode>
)
