export interface LoginRequest {
  username: string;
  password: string;
}

export interface AuthSession {
  token: string;
  tokenType: string;
  expiresIn: number;
}
