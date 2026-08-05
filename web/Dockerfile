# syntax=docker/dockerfile:1
# Kulee Web — multi-stage SPA build.
# Stage 1: build the frontend
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Stage 2: static server
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
