# React Patterns

## Component Design

- **Functional components only** - no class components
- One component per file; file name matches component name
- Co-locate component, tests, and styles in the same directory
- Use `React.FC` sparingly; prefer explicit props typing with destructuring
- Keep components under 150 lines; extract sub-components when exceeding

## State Management

- Use React Query / TanStack Query for all server state
- Use `useState` / `useReducer` for local UI state only
- Avoid global state libraries unless truly needed (context + hooks first)
- Never store derived state; compute it during render

## Props and Composition

- Avoid prop drilling beyond 2 levels; use Context or composition
- Use `children` for layout components; use render props sparingly
- Prefer composition over configuration props (slots pattern)
- Destructure props in the function signature for clarity

## Custom Hooks

- Extract reusable logic into custom hooks (`use` prefix)
- One concern per hook; compose hooks for complex behavior
- Custom hooks should return objects for named access, not tuples (unless simple)
- Co-locate hooks with the feature that uses them

## Performance

- Use `React.memo` only when profiler confirms re-render cost
- Use `useMemo` and `useCallback` only for referential stability or expensive computation
- Never optimize prematurely; measure first with React DevTools Profiler
- Use `lazy()` and `Suspense` for route-level code splitting

## Forms

- Use a form library (React Hook Form, Formik) for non-trivial forms
- Validate with Zod schemas shared between client and server
- Prefer controlled inputs; use uncontrolled only for performance-critical forms

## Side Effects

- `useEffect` should have a single responsibility
- Always specify dependency arrays; avoid suppressing lint warnings
- Cleanup subscriptions, timers, and event listeners in the return function
- Avoid `useEffect` for data fetching; use React Query or a data-fetching hook

## Error Boundaries

- Wrap route-level components in error boundaries
- Provide meaningful fallback UI, not blank screens
- Log errors to a monitoring service in the boundary

## File Structure

```
features/
  auth/
    components/
      LoginForm.tsx
      LoginForm.test.tsx
    hooks/
      useAuth.ts
    api/
      auth.queries.ts
    types.ts
    index.ts
```
