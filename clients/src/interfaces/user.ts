export interface Permission {
  id: number;
  name: string;
}

export interface Role {
  id: number;
  name: string;
  permissions: Permission[];
}

export interface UserInfo {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  roleId: number;
  role: Role;
}

export interface ApiError {
  code: string;
  message: string;
}
