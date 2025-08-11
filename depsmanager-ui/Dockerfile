# Stage 1: Build Angular (18)
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
# wymuś stały output niezależnie od nazwy projektu
RUN npx ng build --configuration production --output-path=dist/browser

# Stage 2: Nginx
FROM nginx:alpine
# podmień domyślny config
COPY nginx.conf /etc/nginx/conf.d/default.conf
# skopiuj gotowe pliki do root, nadpisując welcome page
COPY --from=build /app/dist/browser/ /usr/share/nginx/html/
EXPOSE 80

