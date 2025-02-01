import { TCP } from 'k6/x/tcp';

export default async function () {
  const socket = new TCP()

  await socket.open("127.0.0.1:8080")

  await socket.write("1");

  await socket.done()

  console.log("2")

}
