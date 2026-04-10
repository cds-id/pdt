import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SuggestionList } from './SuggestionList';

describe('SuggestionList', () => {
  it('groups suggestions by kind in order gap/stale/next_step', () => {
    render(
      <SuggestionList
        suggestions={[
          { kind: 'next_step', title: 'n1', detail: 'd', refs: [] },
          { kind: 'gap', title: 'g1', detail: 'd', refs: [] },
          { kind: 'stale', title: 's1', detail: 'd', refs: [] },
        ]}
      />,
    );
    const headings = screen.getAllByRole('heading');
    expect(headings.map((h) => h.textContent)).toEqual(['GAPS', 'STALE WORK', 'NEXT STEPS']);
  });
});
