export interface UserInfo {
  email: string;
  firstName: string;
  id: number;
  lastName: string;
  role: Role;
  roleId: 1;
}

export interface Role {
  name: string;
  permissions: string;
  id: number;
}
