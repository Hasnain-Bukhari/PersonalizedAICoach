# Personalized AI Coach Monorepo

A TypeScript monorepo for building a personalized AI coaching platform with Vue 3 frontend, Express backend, and shared DTOs.

## Project Structure

```
├── /frontend          # Vue 3 + Vite + Tailwind CSS application
│   ├── src/
│   │   ├── assets/    # Static assets (CSS, images)
│   │   ├── components/# Reusable Vue components
│   │   ├── router/    # Vue Router configuration
│   │   ├── views/     # Page components
│   │   └── main.ts    # Application entry point
│   ├── index.html
│   └── package.json
├── /backend           # Node.js + Express + TypeScript API
│   ├── src/
│   │   └── index.ts   # Main server file
│   └── package.json
├── /shared            # Shared DTOs and types (Zod schemas)
│   ├── src/
│   │   ├── user.ts    # User-related DTOs
│   │   ├── coaching-session.ts  # Coaching session DTOs
│   │   └── goal.ts    # Goal-related DTOs
│   └── package.json
├── .eslintrc.cjs     # ESLint configuration
├── .prettierrc.json  # Prettier configuration
├── tsconfig.json     # Root TypeScript configuration
├── pnpm-workspace.yaml  # pnpm workspace definition
└── package.json      # Root package.json with scripts
```

## Prerequisites

- Node.js >= 18.0.0
- pnpm >= 8.0.0

## Installation

```bash
# Install pnpm if not already installed
npm install -g pnpm

# Install dependencies for all packages
pnpm install
```

## Development

### Run all services concurrently

```bash
npm run dev
```

This will start:
- Frontend on http://localhost:5173
- Backend on http://localhost:3001

### Run individual services

```bash
# Frontend only
pnpm --filter frontend dev

# Backend only
pnpm --filter backend dev
```

## Build

```bash
npm run build
```

This will build all packages in the monorepo.

## Scripts Reference

| Command | Description |
|---------|-------------|
| `npm run dev` | Start both frontend and backend concurrently |
| `npm run dev:frontend` | Start only the frontend |
| `npm run dev:backend` | Start only the backend |
| `npm run build` | Build all packages |
| `npm run lint` | Run ESLint with auto-fix |
| `npm run format` | Format code with Prettier |

## Configuration

### Frontend (Vue 3 + Vite + Tailwind CSS)

- **Port**: 5173
- **API URL**: Configured via `VITE_API_URL` environment variable
- **Styling**: Tailwind CSS with PostCSS and Autoprefixer

### Backend (Express + TypeScript)

- **Port**: 3001 (configurable via `PORT` env variable)
- **Entry Point**: `src/index.ts`
- **Features**: CORS enabled, JSON body parsing

### Shared DTOs

- Uses Zod for schema validation
- Exports types and interfaces for both frontend and backend consumption

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Backend server port | 3001 |
| `NODE_ENV` | Environment mode | development |
| `VITE_API_URL` | Frontend API endpoint | http://localhost:3001/api |

## Code Style

This project enforces consistent code style using ESLint and Prettier:

- **ESLint**: TypeScript recommended rules with Vue support
- **Prettier**: 2-space indentation, single quotes, trailing commas

All commits should pass linting before pushing.
