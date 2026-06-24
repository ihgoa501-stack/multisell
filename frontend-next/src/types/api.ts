export interface Result<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

export interface PageResult<T = unknown> {
  code: number;
  message: string;
  data?: T[];
  total: number;
  page: number;
  size: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  roles?: string[];
}
