export interface ServerConfig {
  url: string;
  token: string;
  userToken?: string;
  username?: string;
}

export interface ApiError {
  message: string;
  code?: string;
  status?: number;
}
