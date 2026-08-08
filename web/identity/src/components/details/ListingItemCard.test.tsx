import { render, screen } from '@testing-library/react'
import { Smartphone } from 'lucide-react'
import { describe, expect, it } from 'vitest'
import { ListingItemCard, ListingItemIcon, ListingItemMeta, ListingItemNested } from './ListingItemCard'

describe('ListingItemCard', () => {
  it('exposes reusable theme hooks for rows, icons, meta text, and nested content', () => {
    const { container } = render(
      <ListingItemCard icon={Smartphone} action={<button type="button">Manage</button>}>
        <p>Trusted device</p>
        <ListingItemMeta>Last used today</ListingItemMeta>
        <ListingItemNested>
          <div>Browser</div>
        </ListingItemNested>
      </ListingItemCard>,
    )

    expect(container.querySelector('[data-md-listing-item]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-listing-icon]')).toBeInTheDocument()
    expect(container.querySelector('[data-md-listing-meta]')).toHaveTextContent('Last used today')
    expect(container.querySelector('[data-md-listing-nested]')).toHaveTextContent('Browser')
    expect(screen.getByRole('button', { name: 'Manage' })).toBeInTheDocument()
  })

  it('can render standalone listing icon primitives', () => {
    const { container } = render(
      <ListingItemIcon>
        <Smartphone />
      </ListingItemIcon>,
    )

    expect(container.querySelector('[data-md-listing-icon]')).toBeInTheDocument()
  })
})
