import { TCP } from 'k6/x/tcp';

export default async function () {
  // In this way users can open the socket asyncronosly
  const socket = new TCP()
  console.log(socket.active); // false

  socket.open('127.0.0.1:80', {
                            // default      | possible values
    ipVersion: 0,           // 0            | 4 (IPv4), 6 (IPv6), 0 (both)
    keepAlive: true,        // false        |
    lookup: null,           // dns.lookup() |
    proxy: 'myproxy:3030',  // ''           |
  });
  console.log(socket.active); // still false
}

