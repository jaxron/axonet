<h1 align="center">
  <picture>
    <img width="350" alt="axonet" src="./assets/images/axonet.png">
  </picture>
  <br>
  <a href="https://github.com/jaxron/axonet/blob/main/LICENSE.md">
    <img src="https://img.shields.io/github/license/jaxron/axonet?style=flat-square&color=008ae6">
  </a>
  <a href="https://github.com/jaxron/axonet/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/jaxron/axonet/ci.yml?style=flat-square&color=008ae6">
  </a>
  <a href="https://github.com/jaxron/axonet/issues">
    <img src="https://img.shields.io/github/issues/jaxron/axonet?style=flat-square&color=008ae6">
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
- [🚀 Usage](#-usage)
- [🛠 Configuration](#-configuration)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

# 🚀 Features

Axonet offers features that prioritize flexibility and reliability. Key features include:

- **Custom Client**
  - Make requests to any website using built-in middlewares.
  - Extend the middlewares to suit your needs.
  - Simple request construction using builders
- **Built-in Middlewares:**
  - [Circuit breaker](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker) for fault tolerance.
  - [Retry mechanism](https://learn.microsoft.com/en-us/azure/architecture/patterns/retry) with exponential backoff.
  - [Rate limiting](https://learn.microsoft.com/en-us/azure/architecture/patterns/rate-limiting-pattern) to prevent API throttling.
  - [Single flight](https://pkg.go.dev/golang.org/x/sync/singleflight) to deduplicate concurrent requests.
  - Dynamic proxy rotation for distributed traffic.
  - Cookie-based authentication with rotation.
  - Custom header support for default headers.
- **Easy to Troubleshoot:**
  - Configurable loggers for debugging
  - Detailed error types with root cause

# 📦 Installation

## Installing Main Client

To install the main client package, use the following command:

```bash
go get github.com/jaxron/axonet/pkg/client
```

## Installing Middlewares

Axonet's middlewares are designed to be modular. You can install only the middlewares you need for your project. To install a specific middleware, use the following command pattern:

```bash
go get github.com/jaxron/axonet/middleware/{middleware_name}
```

For example, to install the retry middleware:

```bash
go get github.com/jaxron/axonet/middleware/retry
```

Available middlewares:

- circuitbreaker
- cookie
- header
- proxy
- ratelimit
- retry
- singleflight

Install only the middlewares you plan to use in your project to keep your dependencies minimal.

# 🚀 Usage

Here's a basic example of how to use the client:

First, install the necessary packages:

```bash
go get github.com/jaxron/axonet/pkg/client
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
    // Create a new client with default settings
    c := client.NewClient(
        client.WithMiddleware(retry.New(3, 1*time.Second, 5*time.Second)),
        client.WithMiddleware(singleflight.New()),
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

The client can be configured with various middlewares when using the `NewClient` function. First, install the desired middleware packages:

```bash
go get github.com/jaxron/axonet/middleware/circuitbreaker
go get github.com/jaxron/axonet/middleware/retry
go get github.com/jaxron/axonet/middleware/ratelimit
go get github.com/jaxron/axonet/middleware/singleflight
go get github.com/jaxron/axonet/middleware/proxy
go get github.com/jaxron/axonet/middleware/cookie
go get github.com/jaxron/axonet/middleware/header
```

Then, configure the client:

```go
import (
    "github.com/jaxron/axonet/pkg/client"
    "github.com/jaxron/axonet/middleware/circuitbreaker"
    "github.com/jaxron/axonet/middleware/cookie"
    "github.com/jaxron/axonet/middleware/header"
    "github.com/jaxron/axonet/middleware/proxy"
    "github.com/jaxron/axonet/middleware/ratelimit"
    "github.com/jaxron/axonet/middleware/retry"
    "github.com/jaxron/axonet/middleware/singleflight"
)

client := client.NewClient(
  client.WithMiddleware(circuitbreaker.New(5, 10*time.Second, 30*time.Second)),
    client.WithMiddleware(cookie.New([][]*http.Cookie{cookies1, cookies2})),
    client.WithMiddleware(header.New(http.Header{"User-Agent": {"MyApp/1.0"}})),
    client.WithMiddleware(proxy.New([]*url.URL{proxyURL1, proxyURL2})),
    client.WithMiddleware(ratelimit.New(10, 5)),
    client.WithMiddleware(retry.New(3, 1*time.Second, 5*time.Second)),
    client.WithMiddleware(singleflight.New()),
    client.WithLogger(logger.NewBasicLogger()),
)
```

You can add or remove middlewares as needed for your specific use case. You can also add custom middleware using the `WithMiddleware(middleware Middleware)` option if you need to extend the functionality beyond what is provided.

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
    Header("Content-Type", "application/json").
    MarshalBody(MyRequest{Key: "value"}).
    Result(&myResult).
    Do(context.Background())

if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Println(myResult.Data)
```

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
client.NewRequest().
    MarshalWith(sonic.Marshal).
    UnmarshalWith(sonic.Unmarshal)

// Using go-json
client.NewRequest().
    MarshalWith(json.Marshal).
    UnmarshalWith(json.Unmarshal)
```

The `Do(ctx context.Context)` method executes the request, automatically marshalling the request body and unmarshalling the response if a result is set.

# 🤝 Contributing

This project is open-source and we welcome all contributions from the community! Please feel free to submit a Pull Request.

# 📄 License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

```md
Copyright 2023 jaxron

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
