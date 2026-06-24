import { getStoredUser } from './auth';

export function getCurrentOperator(): string {
  const user = getStoredUser();
  return user?.name || user?.email || 'unknown';
}
