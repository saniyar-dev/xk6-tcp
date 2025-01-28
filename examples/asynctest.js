import { TCP } from 'k6/x/tcp';

export default async function () {
  const socket = new TCP("http://127.0.0.1:8080")


  socket.write("1");


  console.log("2")

}
