import os
import base64
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

def encrypt(key: bytes, insecure: bytes) -> bytes:
    plain_text = insecure.encode()
    iv = os.urandom(16)
    key_bytes = key.encode()

    cipher = Cipher(algorithms.AES(key_bytes), modes.CBC(iv), backend=default_backend())
    encryptor = cipher.encryptor()
    padded_plain_text = PKCS7Padding(plain_text, 16)
    cipher_text = iv + encryptor.update(padded_plain_text) + encryptor.finalize()

    return base64.b64encode(cipher_text).decode()

def decrypt(key: bytes, secure: bytes) -> bytes:
    cipher_text = base64.b64decode(secure)
    key_bytes = key.encode()
    iv = cipher_text[:16]
    cipher_text = cipher_text[16:]

    cipher = Cipher(algorithms.AES(key_bytes), modes.CBC(iv), backend=default_backend())
    decryptor = cipher.decryptor()
    decrypted_text = decryptor.update(cipher_text) + decryptor.finalize()

    padding_length = decrypted_text[-1]
    decrypted_text = decrypted_text[:-padding_length]
    return decrypted_text

def PKCS7Padding(data: bytes, block_size: int) -> bytes:
    padding_length = block_size - (len(data) % block_size)
    padding = bytes([padding_length]) * padding_length
    return data + padding
