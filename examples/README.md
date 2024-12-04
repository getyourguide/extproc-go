# Examples

This directory contains examples of filters built with `extproc-go`. This setup can also be used for integration tests.

```shell
docker compose up --watch
```

```shell
curl http://127.0.0.1:10000 -H "Host: www.example.com"
{
  "headers": {
    "Accept": "*/*",
    "Host": "www.example.com",
    "Method": "GET",
    "User-Agent": "curl/8.7.1",
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000",
    "X-Forwarded-Proto": "http",
    "X-Request-Id": "5b2b4460-cc79-9a95-b276-530d7c46d49d"
  }
}
```
