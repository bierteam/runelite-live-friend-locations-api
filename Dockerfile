# syntax=docker.io/docker/dockerfile:1

FROM node:19-slim as builder
WORKDIR /usr/src/app
COPY package*.json ./
RUN npm ci --omit=dev
COPY . .

FROM node:19-alpine
WORKDIR /usr/src/app
COPY --from=builder /usr/src/app ./
EXPOSE 3000
CMD [ "node", "server.js" ]
