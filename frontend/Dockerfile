FROM node:20-alpine AS builder

WORKDIR /app

RUN npm config set registry https://registry.npmjs.org/

COPY package.json package-lock.json ./

RUN rm -f package-lock.json && npm install

COPY . .

ENV VITE_API_BASE_URL=/api/v1

RUN npm run build

FROM nginx:1.27-alpine AS runtime

RUN apk add --no-cache curl

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chmod -R 755 /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:80/ || exit 1
