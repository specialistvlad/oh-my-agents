# syntax=docker/dockerfile:1

# ─── Stage 1: build ───────────────────────────────────────────────────────────
# The gates (prettier, lint, test) run here, so the image only builds if they
# pass — same contract as the api Dockerfile's check stage.
FROM node:24-alpine AS build

WORKDIR /app

COPY package*.json .npmrc ./
RUN npm ci

COPY eslint.config.js .prettierrc .prettierignore ./
COPY tsconfig.json tsconfig.node.json vite.config.ts vitest.config.ts ./
COPY index.html ./
COPY public/ public/
COPY src/ src/

RUN npm run build
RUN npm run prettier
RUN npm run lint
RUN npm run test

# ─── Stage 2: release ─────────────────────────────────────────────────────────
FROM nginx:alpine AS release

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

ARG COMMIT_HASH=unknown
ARG BUILD_TIME=unknown
RUN printf 'app_name: oh-my-agents-web\ncommit_hash: %s\nbuild_time: %s\n' \
    "${COMMIT_HASH}" "${BUILD_TIME}" > /usr/share/nginx/html/build-info.txt

EXPOSE 39171
