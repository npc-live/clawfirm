import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

// Mock Wails runtime - always present in this Wails/React project
vi.mock('../../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsEmit: vi.fn(),
}))

// Mock localStorage to simulate first launch (no saved state)
const localStorageMock = {
  getItem: vi.fn((key) => {
    if (key === 'pi_settings') return null
    if (key === 'pi_conversations') return '[]'
    return null
  }),
  setItem: vi.fn(),
}
vi.stubGlobal('localStorage', localStorageMock)

// Mock window size for Wails
Object.defineProperty(window, 'size', {
  value: () => ({ width: 1200, height: 800 }),
})

import { App } from './App'

describe('P0: SetupWizard - 首次启动渲染检查', () => {
  beforeEach(() => {
    localStorage.clear()
    // Force first-load state: no saved settings
    vi.mocked(localStorage.getItem).mockImplementation((key) => {
      if (key === 'pi_settings') return null
      if (key === 'pi_conversations') return '[]'
      return null
    })
  })

  it('SetupWizard renders correctly on first launch', async () => {
    render(<App />)

    // Check that the welcome title is visible - indicating setup wizard loaded
    const welcomeTitle = screen.getByText(/欢迎使用 Pi/i)
    expect(welcomeTitle).toBeInTheDocument()

    // Subtitle / description should be present
    const subtitle = screen.getByText(
      /让 AI 成为您值得信赖的智能助手/i,
    )
    expect(subtitle).toBeInTheDocument()
  })
})