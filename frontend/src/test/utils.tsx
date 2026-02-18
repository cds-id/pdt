import React, { PropsWithChildren } from 'react'
import { render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { RenderOptions } from '@testing-library/react'
import { Provider } from 'react-redux'
import { BrowserRouter } from 'react-router-dom'
import { PersistGate } from 'redux-persist/integration/react'
import { store as realStore, persistor, RootState } from '@/application/store'

interface ExtendedRenderOptions extends Omit<RenderOptions, 'queries'> {
  preloadedState?: Partial<RootState>
  store?: typeof realStore
}

export function renderWithProviders(
  ui: React.ReactElement,
  { store = realStore, ...renderOptions }: ExtendedRenderOptions = {}
) {
  function Wrapper({ children }: PropsWithChildren): JSX.Element {
    return (
      <Provider store={store}>
        <PersistGate loading={null} persistor={persistor}>
          <BrowserRouter>{children}</BrowserRouter>
        </PersistGate>
      </Provider>
    )
  }

  // Create user event instance
  const user = userEvent.setup()

  return {
    user,
    store,
    ...render(ui, { wrapper: Wrapper, ...renderOptions })
  }
}

export * from '@testing-library/react'
