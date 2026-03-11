# runelite-live-friend-locations-api

A small HTTP API that stores and returns live friend location updates for the RuneLite plugin.

## Running locally

### Build

```sh
go build -o friend-tracker ./...
```

### Run

```sh
export SHARED_KEY=your-secret
./friend-tracker
```

The API listens on port `3000` by default.

## Docker

Build and run:

```sh
docker build -t runelite-friend-locations-api .
docker run -e SHARED_KEY=your-secret -p 3000:3000 runelite-friend-locations-api
```

## API

### GET /
Returns the current list of tracked locations.

### POST /post
Accepts JSON body with the following fields (name is required):

- `name` (string)
- `waypoint` (object with `x`, `y`, `plane`)
- `x`, `y`, `plane` (int)
- `type`, `title` (string)
- `world` (int)

Include the shared key as the `Authorization` header.

## License

MIT
