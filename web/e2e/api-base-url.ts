export function apiUrl(path: string): string {
  const serverPort = process.env.PLAYWRIGHT_SERVER_PORT ?? '8080';
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;

  return `http://localhost:${serverPort}/api${normalizedPath}`;
}
