import asyncio
import websockets

from aiortc.contrib.signaling import object_from_string, object_to_string

class WebsocketSignaling:
    def __init__(self, server):
        self._server = server
        self._websocket = None

    async def connect(self):
        self._websocket = await websockets.connect(self._server)

    async def close(self):
        if self._websocket is not None and self._websocket.open is True:
            await self.send(None)
            await self._websocket.close()

    async def receive(self):
        try:
            data = await self._websocket.recv()
        except asyncio.IncompleteReadError:
            return
        ret = object_from_string(data)
        if ret == None:
            print("remote host says good bye!")

        return ret

    async def send(self, descr):
        data = object_to_string(descr)
        await self._websocket.send(data + '\n')