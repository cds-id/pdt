import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { StaleWorkTable } from './StaleWorkTable';

describe('StaleWorkTable', () => {
  it('shows empty state when no stale topics', () => {
    render(<StaleWorkTable topics={[]} suggestions={[]} />);
    expect(screen.getByText(/no stale work/i)).toBeInTheDocument();
  });

  it('renders a row per stale topic and attaches matching action', () => {
    render(
      <StaleWorkTable
        topics={[
          {
            anchor: {
              card_key: 'A-1',
              title: 'Login',
              status: 'In Progress',
              assignee: '',
              content: '',
              updated_at: '',
            },
            messages: [],
            commits: [],
            stale: true,
            days_idle: 9,
          },
          {
            anchor: {
              card_key: 'A-2',
              title: 'Shipped',
              status: 'Done',
              assignee: '',
              content: '',
              updated_at: '',
            },
            messages: [],
            commits: [],
            stale: false,
            days_idle: 0,
          },
        ]}
        suggestions={[{ kind: 'stale', title: 't', detail: 'Ping owner', refs: ['jira:A-1'] }]}
      />,
    );
    expect(screen.getByText('A-1')).toBeInTheDocument();
    expect(screen.queryByText('A-2')).not.toBeInTheDocument();
    expect(screen.getByText('Ping owner')).toBeInTheDocument();
    expect(screen.getByText('9')).toBeInTheDocument();
  });
});
