import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import AppShell from './components/layout/AppShell';
import Dashboard from './pages/Dashboard';
import Applications from './pages/Applications';
import Infrastructure from './pages/Infrastructure';
import Compliance from './pages/Compliance';
import Scaffold from './pages/Scaffold';
import NotFound from './pages/NotFound';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 10000,
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<AppShell />}>
            <Route index element={<Dashboard />} />
            <Route path="apps" element={<Applications />} />
            <Route path="infra" element={<Infrastructure />} />
            <Route path="compliance" element={<Compliance />} />
            <Route path="scaffold" element={<Scaffold />} />
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
