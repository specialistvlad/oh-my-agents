/**
 * Removes every handler from a socket before it is abandoned.
 *
 * A close event arrives asynchronously, so a socket that still has handlers
 * attached can act on behalf of a client that has already moved on — and a
 * reconnect triggered that way leaves two live sockets delivering every
 * event twice. Detaching first is what makes abandoning one final.
 */
export function detach(socket: WebSocket): void {
  socket.onopen = null;
  socket.onmessage = null;
  socket.onclose = null;
  socket.onerror = null;
}
