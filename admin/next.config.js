/** @type {import('next').NextConfig} */
const nextConfig = {
    experimental: {
        serverActions: {
            allowedOrigins: ["185.204.168.179", "185.226.116.124", "localhost"]
        }
    }
}

module.exports = nextConfig
