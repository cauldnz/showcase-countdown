"""Expose a loopback-only port on the LAN, without admin rights.

Podman on Windows publishes container ports on 127.0.0.1 only, so a stick on
the LAN cannot reach the dev broker. This relay listens on a LAN address and
forwards each connection to the loopback listener. Not for the event: the
router runs Mosquitto natively there.

Usage:
    python scripts/lan_relay.py 192.168.1.201:1883 127.0.0.1:1883
"""

import socket
import sys
import threading


def pump(src, dst):
    try:
        while True:
            data = src.recv(65536)
            if not data:
                break
            dst.sendall(data)
    except OSError:
        pass
    finally:
        for s in (src, dst):
            try:
                s.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            s.close()


def serve(listen, target):
    lhost, lport = listen.rsplit(":", 1)
    thost, tport = target.rsplit(":", 1)
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    server.bind((lhost, int(lport)))
    server.listen(32)
    print("relay %s -> %s" % (listen, target), flush=True)
    while True:
        client, peer = server.accept()
        try:
            upstream = socket.create_connection((thost, int(tport)), timeout=5)
        except OSError as exc:
            print("upstream refused for %s:%d: %s" % (peer[0], peer[1], exc), flush=True)
            client.close()
            continue
        upstream.settimeout(None)  # the connect timeout must not apply to idle reads
        client.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        upstream.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
        print("client %s:%d connected" % peer, flush=True)
        threading.Thread(target=pump, args=(client, upstream), daemon=True).start()
        threading.Thread(target=pump, args=(upstream, client), daemon=True).start()


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit(__doc__)
    serve(sys.argv[1], sys.argv[2])
