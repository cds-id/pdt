# React Vite Templates

This template provides a robust starting point for React applications with modern tooling and best practices.

## Features

- ⚡️ **Vite** - Lightning fast development & building
- ⚛️ **React 18** - Latest React features
- 🔧 **TypeScript** - Type safety
- 🎨 **TailwindCSS 3** - Utility-first CSS
- 📈 **Redux Toolkit** - State management
- 💾 **Redux Persist** - State persistence
- 🔄 **Axios** - API requests
- 🚦 **React Router** - Routing
- 📝 **React Hook Form** - Form handling
- ✅ **Yup** - Form validation
- 🎭 **HeadlessUI** - Accessible components
- 🧪 **Vitest** - Testing
- 📋 **Commitlint** - Commit conventions
- 🐶 **Husky** - Git hooks
- 📦 **Release-it** - Release management
- 🔍 **ESLint/Prettier** - Code quality

## Project Structure

```
src/
├── application/        # Application layer
│   ├── hooks/         # Custom hooks
│   └── store/         # Redux store setup
├── domain/            # Domain layer
│   ├── interfaces/    # TypeScript interfaces
│   ├── repositories/  # Data access
│   └── useCases/      # Business logic
├── infrastructure/    # Infrastructure layer
│   ├── api/          # API setup
│   └── slices/       # Redux slices
└── presentation/     # Presentation layer
    ├── components/   # Reusable components
    ├── pages/        # Route pages
    └── routes/       # Router setup
```

## Getting Started

1. Use this template:
```bash
npx degit cds-id/vite-tailwind-boilerplate my-project
```

2. Setup project:
```bash
cd my-project
npm run setup
```

3. Install dependencies:
```bash
npm install
```

4. Start development:
```bash
npm run dev
```

## Development Guide

### Creating New Features

Follow Domain-Driven Development pattern:

1. Define interfaces in `domain/interfaces`
2. Create repository in `domain/repositories`
3. Implement use cases in `domain/useCases`
4. Add Redux slice in `infrastructure/slices`
5. Create components in `presentation/components`

### State Management

Use Redux Toolkit with persist:
```typescript
// Create slice
const slice = createSlice({...})

// Use in components
const data = useAppSelector(state => state.slice.data)
const dispatch = useAppDispatch()
```

### API Requests

Use axios instance with interceptors:
```typescript
import api from '@/infrastructure/api/axios'

const data = await api.get('/endpoint')
```

### Testing

Write tests for components and logic:
```typescript
import { renderWithProviders } from '@/test/utils'

describe('Component', () => {
  it('renders', () => {
    renderWithProviders(<Component />)
  })
})
```

### Git Workflow

1. Create feature branch
2. Make changes
3. Commit with conventional commits:
```bash
npm run commit
```
4. Push changes:
```bash
npm run push
```
5. Create release:
```bash
npm run release
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run test` - Run tests
- `npm run lint` - Lint code
- `npm run commit` - Create conventional commit
- `npm run release` - Create new release
