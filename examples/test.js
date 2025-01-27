import { TCP } from 'k6/x/tcp';

export default async function () {
  const socket = await TCP.open('127.0.0.1:80', {
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
