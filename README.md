# xk6-tcp

This extension adds the ability to open and write on TCP sockets in the term of new HTTP Design.

See this [link](https://github.com/grafana/k6/blob/master/docs/design/018-new-http-api.md#sockets) for more details.

## Requirements

- [Goland 1.20+](https://go.dev/)
- [Git](https://git-scm.com/)
- [xk6](https://github.com/grafana/xk6) (`go install go.k6.io/xk6/cmd/xk6@latest`)

## Getting started

1. Build the k6 binary:
`make build`

2. Run an example:
`./k6 run ./examples/test.js`

## Usage/Examples

```javascript
import { TCP } from 'k6/x/tcp';

export default async function () {
  const socket = await TCP.open('192.168.1.1:80', {
                            // default      | possible values
    ipVersion: 0,           // 0            | 4 (IPv4), 6 (IPv6), 0 (both)
    keepAlive: true,        // false        |
    lookup: null,           // dns.lookup() |
    proxy: 'myproxy:3030',  // ''           |
  });
  console.log(socket.active); // true

  // Writing directly to the socket.
  // Requires TextEncoder implementation, otherwise typed arrays can be used as well.
  await socket.write(new TextEncoder().encode('GET / HTTP/1.1\r\n\r\n'));

  // And reading...
  socket.on('data', data => {
    console.log(`received ${data}`);
    socket.close();
  });

  await socket.done();
}
```

## 🔗 Links
[![linkedin](https://img.shields.io/badge/linkedin-0A66C2?style=for-the-badge&logo=linkedin&logoColor=white)](https://www.linkedin.com/in/saniyar-karami-818771231/)
[![twitter](https://img.shields.io/badge/twitter-1DA1F2?style=for-the-badge&logo=twitter&logoColor=white)](https://twitter.com/)

