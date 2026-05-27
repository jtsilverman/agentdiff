import { vi } from 'vitest';

export const mockUsePathname = vi.fn(() => '/');
export const mockUseParams = vi.fn(() => ({}));
export const mockUseRouter = vi.fn(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  back: vi.fn(),
  prefetch: vi.fn(),
}));
export const mockUseSearchParams = vi.fn(
  () => new URLSearchParams() as unknown as ReturnType<typeof vi.fn>,
);

vi.mock('next/navigation', () => ({
  usePathname: mockUsePathname,
  useParams: mockUseParams,
  useRouter: mockUseRouter,
  useSearchParams: mockUseSearchParams,
}));
