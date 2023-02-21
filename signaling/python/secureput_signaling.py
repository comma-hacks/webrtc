import asyncio
from aiortc.contrib.signaling import RTCSessionDescription, RTCIceCandidate, candidate_from_sdp, candidate_to_sdp
import json

from .websocket_signaling import WebsocketSignaling
from . import aes
from .app import App

class SecureputSignaling(WebsocketSignaling):
    def __init__(self, server):
        super().__init__(server)
        self._app = App()
        
        if not self._app.paired():
            self._app.gen_pair_info()

    async def connect(self):
        await super().connect()
        await self.sendIdentity()

    async def close(self):
        if self._websocket is not None and self._websocket.open is True:
            await self._websocket.close()

    def secret(self):
        return self._app.config["deviceSecret"]

    def encrypt(self, data):
        return aes.encrypt(self.secret(), data)

    def decrypt(self, data):
        return aes.decrypt(self.secret(), data)

    def __object_from_string(self, message_str):
        message = json.loads(message_str)

        if message["type"] == "wrapped":
            message = json.loads(self.decrypt(message["payload"]["data"]))

        return message

    def __object_to_string(self, obj):
        if isinstance(obj, RTCSessionDescription):
            message = self.forwardWrap({
                "type": "SessionDescription",
                "payload": {"sdp": obj.sdp, "type": obj.type}
            })
        elif isinstance(obj, RTCIceCandidate):
            message = self.forwardWrap({
                "type": "IceCandidate",
                "payload": {
                    "sdp": "candidate:" + candidate_to_sdp(obj),
                    "sdpMid": obj.sdpMid,
                    "sdpMLineIndex": obj.sdpMLineIndex,
                }
            })
        else:
            message = obj

        return json.dumps(message, sort_keys=True)

    async def send(self, descr):
        data = self.__object_to_string(descr)
        await self._websocket.send(data + '\n')

    async def receive(self):
        try:
            data = await self._websocket.recv()
        except asyncio.IncompleteReadError:
            return
        ret = self.__object_from_string(data)
        if ret == None:
            print("remote host says good bye!")
        elif isinstance(ret, dict):
            if ret['type'] == 'claim':
                await self.claim(ret['payload']['account'])
            else:
                if ret["type"] == "SessionDescription":
                    sdp = ret["payload"]["sdp"]
                    type = ret["payload"]["type"]
                    return RTCSessionDescription(sdp=sdp, type=type)
                elif ret["type"] == "IceCandidate":
                    candidate = candidate_from_sdp(ret["payload"]["sdp"].split(":", 1)[1])
                    candidate.sdpMid = ret["payload"]["sdpMid"]
                    candidate.sdpMLineIndex = ret["payload"]["sdpMLineIndex"]
                    return candidate
        return ret

    async def sendIdentity(self):
        await self._websocket.send(json.dumps({
            "type": "identify-target",
            "payload": {
                "name": self._app.config["deviceName"],
                "device": self._app.config["deviceUUID"],
                "account": self._app.config["accountUUID"],
                "metadata": {
                    "isRobot": True
                }
            }
        }))

    async def claim(self, account):
        self._app.config["accountUUID"] = account
        print("claimed to account %s" % account)
        await self.sendIdentity()

    def forwardWrap(self, json_data: dict) -> str:
        msg = {
            'to': self._app.config["accountUUID"],
            'type': 'forward-wrapped',
        }
        # Encrypt the plaintext
        ciphertext = self.encrypt(json.dumps(json_data))
        
        # Install the ciphertext into the payload
        msg['body'] = ciphertext
        
        return msg
