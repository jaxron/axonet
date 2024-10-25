<h1 align="center">
  <picture>
    <img width="350" alt="axonet" src="./assets/images/axonet.png">
  </picture>
  <br>
  <a href="https://github.com/jaxron/axonet/blob/main/LICENSE.md">
    <img src="https://img.shields.io/github/license/jaxron/axonet?style=flat-square&color=4a92e1">
  </a>
  <a href="https://github.com/jaxron/axonet/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/jaxron/axonet/ci.yml?style=flat-square&color=4a92e1">
  </a>
  <a href="https://github.com/jaxron/axonet/issues">
    <img src="https://img.shields.io/github/issues/jaxron/axonet?style=flat-square&color=4a92e1">
  </a>
</h1>

<p align="center">
  <em><b>axonet</b> is a custom HTTP client library for <a href="https://golang.org/">Go</a>, offering middleware options for enhanced request handling.</em>
</p>

---

> [!NOTE]
> This library is in early development. While it's functional for projects, please be aware that significant changes may occur as we refine the project based on feedback.

# 📚 Table of Contents

- [🚀 Features](#-features)
- [📦 Installation](#-installation)
- [🧩 Middlewares](#-middlewares)
- [🚀 Usage](#-usage)
- [🛠 Configuration](#-configuration)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)
- [❓ FAQ](#-faq)

# 🚀 Features

Axonet offers features that prioritize flexibility and reliability. Key features include:

- **Enhanced HTTP Client**
  - Make requests to any website using built-in middlewares.
  - Extend the middlewares to suit your needs.
  - Simple request construction using builders.
  - Built on top of Go's standard `http.Client`.
- **Built-in Middlewares:**
  - Circuit breaker for fault tolerance.
  - Retry mechanism with exponential backoff.
  - Rate limiting to prevent API throttling.
  - Single flight to deduplicate concurrent requests.
  - Dynamic proxy rotation for distributed traffic.
  - Redis for response caching.
  - [And more!](#available-middlewares)
- **Easy to Troubleshoot:**
  - Configurable loggers for debugging
  - Detailed error types with root cause

# 📦 Installation

To install the main client package, use the following command:

```bash
go get github.com/jaxron/axonet
```

# 🧩 Middlewares

Axonet provides a variety of middlewares to augment your HTTP client's functionality. You have the flexibility to selectively install and utilize only the middlewares that are essential for your project. If you require capabilities beyond the provided options, you can create custom middleware and integrate it seamlessly using the `WithMiddleware(priority int, middleware Middleware)` client option.

## Available Middlewares

| Middleware      | Description                                                                                                                                   | Recommended Priority | Source                                                                         |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | -------------------- | ------------------------------------------------------------------------------ |
| Circuit Breaker | Implements fault tolerance using the [circuit breaker](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker) pattern | 6                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/circuitbreaker) |
| Retry           | Provides [retry mechanism](https://learn.microsoft.com/en-us/azure/architecture/patterns/retry) with exponential backoff                      | 5                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/retry)          |
| Single Flight   | Deduplicates concurrent identical requests                                                                                                    | 4                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/singleflight)   |
| Redis           | Provides response caching using Redis                                                                                                         | 3                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/redis)          |
| Rate Limit      | Implements [rate limiting](https://learn.microsoft.com/en-us/azure/architecture/patterns/rate-limiting-pattern) to prevent API throttling     | 2                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/ratelimit)      |
| Header          | Adds custom headers to requests                                                                                                               | 1                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/header)         |
| Cookie          | Manages cookie-based authentication with rotation                                                                                             | 1                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/cookie)         |
| Proxy           | Enables dynamic proxy rotation for distributed traffic                                                                                        | 1                    | [Source](https://github.com/jaxron/axonet/tree/main/middleware/proxy)          |

## Installing Middlewares

To install a specific middleware, use the following command pattern:

```bash
go get github.com/jaxron/axonet/middleware/{middleware_name}
```

For example, to install the retry middleware:

```bash
go get github.com/jaxron/axonet/middleware/retry
```

# 🚀 Usage

Here's a basic example of how to use the client:

First, install the necessary packages:

```bash
go get github.com/jaxron/axonet
go get github.com/jaxron/axonet/middleware/retry
go get github.com/jaxron/axonet/middleware/singleflight
```

Then, use the client in your code:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/jaxron/axonet/middleware/retry"
    "github.com/jaxron/axonet/middleware/singleflight"
    "github.com/jaxron/axonet/pkg/client"
    "github.com/jaxron/axonet/pkg/client/logger"
)

func main() {
    // Create a new client with middlewares
    c := client.NewClient(
        client.WithMiddleware(2, retry.New(3, 1*time.Second, 5*time.Second)),
        client.WithMiddleware(1, singleflight.New()),
        client.WithLogger(logger.NewBasicLogger()),
    )

    // Make a request
    resp, err := c.NewRequest().
        Method(http.MethodGet).
        URL("https://api.example.com/data").
        Do(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    // Process the response...
}
```

# 🛠 Configuration

## Client Configuration

You can configure the client with various middlewares when using the `NewClient` function. Here's an example of how to configure multiple middlewares:

```go
import (
    "github.com/jaxron/axonet/pkg/client"
    "github.com/jaxron/axonet/middleware/circuitbreaker"
    "github.com/jaxron/axonet/middleware/retry"
    "github.com/jaxron/axonet/middleware/singleflight"
    "github.com/jaxron/axonet/middleware/redis"
    "github.com/jaxron/axonet/middleware/ratelimit"
    "github.com/jaxron/axonet/middleware/proxy"
    "github.com/jaxron/axonet/middleware/cookie"
    "github.com/jaxron/axonet/middleware/header"
)

c := client.NewClient(
    client.WithMiddleware(6, circuitbreaker.New(5, 10*time.Second, 30*time.Second)),
    client.WithMiddleware(5, retry.New(3, 1*time.Second, 5*time.Second)),
    client.WithMiddleware(4, singleflight.New()),
    client.WithMiddleware(3, redis.New(rueidisClient, 5*time.Minute)),
    client.WithMiddleware(2, ratelimit.New(10, 5)),
    client.WithMiddleware(1, proxy.New([]*url.URL{proxyURL1, proxyURL2})),
    client.WithMiddleware(1, cookie.New([][]*http.Cookie{cookies1, cookies2})),
    client.WithMiddleware(1, header.New(http.Header{"User-Agent": {"MyApp/1.0"}})),
    client.WithLogger(logger.NewBasicLogger()),
)
```

## Request Configuration

Individual requests can be configured using the `Request` builder:

```go
type MyRequest struct {
    Key string `json:"key"`
}

type MyResponse struct {
    Data string `json:"data"`
}

var myResult MyResponse

resp, err := c.NewRequest().
    Method(http.MethodPost).
    URL("https://api.example.com/data").
    Query("param1", "value1").
    MarshalBody(MyRequest{Key: "value"}).
    Result(&myResult).
    Do(context.Background())
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Println(myResult.Data)
```

The `Do(ctx context.Context)` method executes the request, automatically marshalling the request body and unmarshaling the response if a result is set.

About some of request configuration:

- `MarshalBody(interface{})`: Automatically marshals the provided struct.
- `MarshalWith(MarshalFunc)`: Sets a custom marshal function for the request body.
- `UnmarshalWith(UnmarshalFunc)`: Sets a custom unmarshal function for the response.
- `Result(interface{})`: Sets the struct to unmarshal the response into.

You can use high-performance JSON libraries like [Sonic](https://github.com/bytedance/sonic) or [go-json](https://github.com/goccy/go-json) for faster marshaling and unmarshaling:

```go
import (
    "github.com/bytedance/sonic"
    "github.com/goccy/go-json"
)

// Using Sonic
c.NewRequest().
    MarshalWith(sonic.Marshal).
    UnmarshalWith(sonic.Unmarshal)

// Using go-json
c.NewRequest().
    MarshalWith(json.Marshal).
    UnmarshalWith(json.Unmarshal)
```

# 🤝 Contributing

This project is open-source and we welcome all contributions from the community! Please feel free to submit a Pull Request.

# 📄 License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

```md
Copyright 2024 jaxron

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

# ❓ FAQ

<details>
  <summary><b>Why use axonet over the standard go HTTP package?</b></summary>
  <p>
  Axonet offers several advantages over the standard Go HTTP package:

1. **Middleware System**: Axonet provides a flexible middleware system for easy integration of features like retry logic, circuit breaking, proxies, cookies, redis and rate limiting. With the standard package, you would need to implement these yourself.

2. **Request Building**: Axonet's request builder makes it simpler to construct complex requests, compared to manually setting up requests with the standard package.

3. **Error Handling**: Axonet offers more detailed error types, making it easier to handle specific error scenarios compared to the standard package's more generic errors.

4. **Automatic Marshaling/Unmarshaling**: Axonet can automatically handle JSON marshaling and unmarshaling, which you'd need to do manually with the standard package.

5. **Configurable Logging**: Easy-to-use logging system for debugging, which isn't provided out-of-the-box with the standard package.

6. **Performance Optimizations**: Axonet includes optimizations like request deduplication (via singleflight) and caching (via redis), which aren't available in the standard package.

While the standard Go HTTP package is suitable for most cases, axonet provides a higher-level abstraction that's particularly well-suited for interacting with complex APIs, saving development time and reducing boilerplate code. It is also entirely up to your project's requirements.

  </p>
</details>
