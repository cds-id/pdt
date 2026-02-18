import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/utils'
import { Button } from './Button'

describe('Button', () => {
  it('handles click events', async () => {
    const handleClick = vi.fn()
    const { user } = renderWithProviders(
      <Button onClick={handleClick}>Click me</Button>
    )

    await user.click(screen.getByRole('button'))
    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('handles keyboard interaction', async () => {
    const handleClick = vi.fn()
    const { user } = renderWithProviders(
      <Button onClick={handleClick}>Click me</Button>
    )

    const button = screen.getByRole('button')
    await user.tab() // Tab to the button
    expect(button).toHaveFocus()

    await user.keyboard('[Enter]') // Press Enter
    expect(handleClick).toHaveBeenCalledTimes(1)
  })
})
