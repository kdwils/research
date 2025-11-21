# Service Health Monitor - Dashboard Frontend

React + TypeScript frontend for the Kubernetes Service Health Monitor.

## Development

### Prerequisites

- Node.js 18+ and npm

### Install Dependencies

```bash
npm install
```

### Run Development Server

```bash
npm run dev
```

The dev server will proxy API and WebSocket requests to `http://localhost:8080` (the dashboard backend).

### Build for Production

```bash
npm run build
```

Built files will be output to `dist/` directory, which the Go dashboard server will serve.

## Features

- **Real-time Updates**: WebSocket integration for live health status changes
- **Dashboard**: Overview of all health checks with summary statistics
- **Detail View**: Comprehensive view of individual health checks
- **Filtering**: Search and filter health checks by name, namespace, or status
- **Uptime Charts**: Visual representation of uptime trends
- **Responsive**: Mobile-friendly design using Tailwind CSS

## Project Structure

```
src/
├── components/          # React components
│   ├── Dashboard.tsx    # Main dashboard with health checks list
│   ├── HealthCheckDetail.tsx  # Detailed health check view
│   └── HealthBadge.tsx  # Health status badge component
├── hooks/              # Custom React hooks
│   └── useWebSocket.ts # WebSocket connection and updates
├── types/              # TypeScript types
│   └── healthcheck.ts  # HealthCheck API types
├── utils/              # Utilities
│   └── api.ts          # API client functions
├── App.tsx             # Main app component with routing
├── main.tsx            # Entry point
└── index.css           # Global styles (Tailwind)
```

## Technologies

- **React 18**: UI framework
- **TypeScript**: Type safety
- **React Router**: Client-side routing
- **Tailwind CSS**: Utility-first CSS
- **Recharts**: Charts for uptime visualization
- **Vite**: Build tool and dev server
