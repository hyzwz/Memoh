export interface BaseMCPConnection {
  type: string
  name: string
}

export interface StdioMCPConnection extends BaseMCPConnection {
  type: 'stdio'
  command: string
  args: string[]
  env: Record<string, string>
  cwd: string
}

export interface BaseHTTPMCPConnection extends BaseMCPConnection {
  url: string
  headers: Record<string, string>
}

export interface HTTPMCPConnection extends BaseHTTPMCPConnection {
  type: 'http'
}

export interface SSEMCPConnection extends BaseHTTPMCPConnection {
  type: 'sse'
}

export type MCPConnection =
  | StdioMCPConnection
  | HTTPMCPConnection
  | SSEMCPConnection


export interface MCPListItem{
  id: string;
  type: string;
  name: string;
  config: {
    cwd: string;
    env: Record<string, string>;
    args: string[];
    type: string;
    command: string;
  };
  active: boolean;
  user: string;
  createdAt: string;
  updatedAt: string;
}