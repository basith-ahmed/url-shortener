This is a simple URL shortener implemented in Go using the Fiber framework.

## API Endpoints

* **Resolve a Shortened URL:**

  `GET /:url`

  Example: `http://localhost:3000/example-short-url`

* **Shorten a URL:**

  `GET /api/v1`

  Example: `http://localhost:3000/api/v1?url=https://example.com/long-url-to-shorten`
