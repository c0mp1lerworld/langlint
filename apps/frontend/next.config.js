/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  typedRoutes: true,
  // Necesario para la imagen Docker (apps/frontend/Dockerfile): genera un
  // runtime autocontenido en .next/standalone.
  output: 'standalone',
};

module.exports = nextConfig;
