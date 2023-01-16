import random

runes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

def rand_seq(n):
    b = []
    for i in range(n):
        b.append(random.choice(runes))
    return "".join(b)

def generate_secret_key():
    return rand_seq(32)
