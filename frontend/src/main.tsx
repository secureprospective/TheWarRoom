import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import { NativeStartup } from './app/control/NativeStartup'
import { plainBindings } from './app/data/plainFailure'

// Before any screen calls Go: an offline failure reads as one plain sentence everywhere.
plainBindings((window as { go?: unknown }).go)

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <NativeStartup/>
    </React.StrictMode>
)
