from vkaudiotoken import get_kate_token, get_vk_official_token
import sys
import json

login = sys.argv[1] # your vk login, e-mail or phone number
password = sys.argv[2] # your vk password
token = None
if len(sys.argv) > 3 :
    token = sys.argv[3]

# print tokens and corresponding user-agent headers
print(get_kate_token(login, password, non_refreshed_token=token).get("token"))